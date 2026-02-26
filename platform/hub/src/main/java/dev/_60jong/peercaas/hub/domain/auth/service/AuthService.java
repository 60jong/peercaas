package dev._60jong.peercaas.hub.domain.auth.service;

import dev._60jong.peercaas.common.util.KeyGenerator;
import dev._60jong.peercaas.hub.domain.auth.controller.api.request.ReissueRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.request.ResetPasswordRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.request.NormalSigninRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.request.NormalSignupRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.response.GetKeyResponse;
import dev._60jong.peercaas.hub.domain.auth.controller.api.response.TokenResponse;
import dev._60jong.peercaas.hub.domain.auth.util.JwtProvider;
import dev._60jong.peercaas.hub.domain.member.model.AccountType;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import dev._60jong.peercaas.hub.domain.member.model.vo.MemberParam;
import dev._60jong.peercaas.hub.global.exception.BaseException;
import dev._60jong.peercaas.hub.domain.auth.util.PasswordEncryptor;
import dev._60jong.peercaas.hub.infra.cache.service.CacheService;
import dev._60jong.peercaas.hub.domain.member.service.MemberService;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import static dev._60jong.peercaas.hub.domain.agent.config.AgentConstants.CLIENT_AGENT_KEY_NAME;
import static dev._60jong.peercaas.hub.domain.agent.config.AgentConstants.WORKER_AGENT_KEY_NAME;
import static dev._60jong.peercaas.hub.global.exception.auth.AuthExceptionCode.*;
import static dev._60jong.peercaas.hub.global.exception.member.MemberExceptionCode.ENTITY_NOT_FOUND;

@Transactional
@Service
public class AuthService {
    private static final String CACHE_NAME = "refresh_tokens";

    private final MemberService memberService;

    private final PasswordEncryptor passwordEncryptor;
    private final JwtProvider jwtProvider;
    private final CacheService cacheService;
    private final Long remainingDue;

    public AuthService(
            MemberService memberService,
            PasswordEncryptor passwordEncryptor,
            JwtProvider jwtProvider,
            CacheService cacheService,
            @Value("${jwt.remaining-due}") Long remainingDue
    ) {
        this.memberService = memberService;
        this.passwordEncryptor = passwordEncryptor;
        this.jwtProvider = jwtProvider;
        this.cacheService = cacheService;
        this.remainingDue = remainingDue;
    }

    /**
     * Client Agent의 Key를 발급
     */
    public GetKeyResponse issueClientKeyByMemberId(Long memberId) {
        if (!memberService.existsById(memberId)) {
            throw new BaseException(ENTITY_NOT_FOUND, "Member not found");
        }

        String key = KeyGenerator.generate();
        cacheService.put(CLIENT_AGENT_KEY_NAME, key, memberId);

        return new GetKeyResponse(key);
    }

    /**
     * Worker Agent의 Key를 발급
     */
    public GetKeyResponse issueWorkerKeyByMemberId(Long memberId) {
        if (!memberService.existsById(memberId)) {
            throw new BaseException(ENTITY_NOT_FOUND, "Member not found");
        }

        String key = KeyGenerator.generate();
        cacheService.put(WORKER_AGENT_KEY_NAME, key, memberId);

        return new GetKeyResponse(key);
    }
    /**
     * 회원가입
     */
    @Transactional
    public TokenResponse signup(NormalSignupRequest request) {
        // 1. 파라미터 검사
        validateBeforeSignup(request);

        // 2. 회원 저장
        String encodedPw = passwordEncryptor.encrypt(request.getPassword());
        Member member = memberService.createMember(
                MemberParam.builder()
                        .nickname(request.getNickname())
                        .email(request.getEmail())
                        .password(encodedPw)
                        .accountType(AccountType.NORMAL)
                        .build()
        );

        // 3. 토큰 발급 및 반환
        return issueTokens(member.getId(), member.getNickname());
    }

    /**
     * 로그인
     */
    @Transactional
    public TokenResponse signin(NormalSigninRequest request) {
        validateBeforeSignin(request);

        Member member = memberService.findByEmail(request.getEmail());
        return issueTokens(member.getId(), member.getNickname());
    }

    /**
     * 토큰 재발급 (Reissue)
     */
    @Transactional
    public TokenResponse reissue(ReissueRequest request) {
        String refreshToken = request.getRefreshToken();

        // 1. Refresh Token 검증
        if (!jwtProvider.validateToken(refreshToken)) {
            throw new RuntimeException("유효하지 않은 Refresh Token입니다.");
        }

        // 2. 저장소에 있는 토큰인지 확인 (탈취/로그아웃 된 토큰 방지)
        Long memberId = jwtProvider.getMemberId(refreshToken);
        String savedToken = cacheService.get(CACHE_NAME, String.valueOf(memberId), String.class)
                .orElseThrow(() -> new BaseException(EXPIRED_REFRESH_TOKEN, "재로그인이 필요합니다."));

        if (!savedToken.equals(refreshToken)) {
            throw new RuntimeException("토큰 정보가 일치하지 않습니다.");
        }

        Member member = memberService.findById(memberId);

        // 3. Access Token 무조건 재발급
        String newAccessToken = jwtProvider.createAccessToken(memberId);

        // 4. Refresh Token 갱신 판단 로직
        String newRefreshToken = refreshToken; // 기본은 기존 것 유지
        long remainingTime = jwtProvider.getRemainingTime(refreshToken);

        // 만료까지 remainingDue 미만으로 남았다면 새로 발급
        if (remainingTime < remainingDue) {
            newRefreshToken = jwtProvider.createRefreshToken(memberId);
            // 저장소 업데이트
            cacheService.put(CACHE_NAME, String.valueOf(memberId), newRefreshToken);
        }

        return new TokenResponse(newAccessToken, newRefreshToken, member.getNickname());
    }


    /**
     * 비밀번호 재발급 (일반 회원용)
     */
    @Transactional
    public void resetPassword(ResetPasswordRequest request) {
        String newEncryptedPassword = passwordEncryptor.encrypt(request.getNewPassword());

        // Dirty Checking
        Member member = memberService.findById(request.getMemberId());
        member.resetPassword(newEncryptedPassword);
    }

    // ============================================
    // ----- Helper Method -----
    // ============================================
    private TokenResponse issueTokens(Long memberId, String nickname) {
        String accessToken = jwtProvider.createAccessToken(memberId);
        String refreshToken = jwtProvider.createRefreshToken(memberId);

        // Refresh Token 저장
        cacheService.put(CACHE_NAME, String.valueOf(memberId), refreshToken);

        return new TokenResponse(accessToken, refreshToken, nickname);
    }

    private void validateBeforeSignup(NormalSignupRequest request) {
        // 1. email 존재 확인
        if (memberService.existsByEmail(request.getEmail())) {
            throw new BaseException(DUPLICATE_EMAIL, "이미 존재하는 Email입니다.");
        }
    }

    private void validateBeforeSignin(NormalSigninRequest request) {
        if (request.getEmail() == null || request.getPassword() == null) {
            throw new BaseException(INVALID_MEMBER_INFO, "이메일과 비밀번호를 모두 입력해주세요.");
        }
        // 1. email 존재 확인 (NotNull, 존재하지 않을 경우 exception 발생됨)
        Member member = memberService.findByEmail(request.getEmail());

        // 2. password 일치 확인
        if (!passwordEncryptor.match(request.getPassword(), member.getPassword())) {
            throw new BaseException(INVALID_PASSWORD, "비밀번호가 일치하지 않습니다.");
        }
    }
}
