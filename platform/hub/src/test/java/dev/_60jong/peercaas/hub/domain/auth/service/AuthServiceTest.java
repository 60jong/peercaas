package dev._60jong.peercaas.hub.domain.auth.service;

import dev._60jong.peercaas.hub.domain.auth.controller.api.request.ReissueRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.request.NormalSigninRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.request.NormalSignupRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.response.TokenResponse;
import dev._60jong.peercaas.hub.domain.auth.util.JwtProvider;
import dev._60jong.peercaas.hub.domain.auth.util.PasswordEncryptor;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import dev._60jong.peercaas.hub.domain.member.service.MemberService;
import dev._60jong.peercaas.hub.global.exception.BaseException;
import dev._60jong.peercaas.hub.infra.cache.service.CacheService;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.test.util.ReflectionTestUtils;

import java.util.Optional;

import static dev._60jong.peercaas.hub.global.exception.auth.AuthExceptionCode.EXPIRED_REFRESH_TOKEN;
import static dev._60jong.peercaas.hub.global.exception.auth.AuthExceptionCode.INVALID_PASSWORD;
import static dev._60jong.peercaas.hub.global.exception.member.MemberExceptionCode.ENTITY_NOT_FOUND;
import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.BDDMockito.given;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;

@ExtendWith(MockitoExtension.class)
class AuthServiceTest {

    private AuthService authService;

    @Mock private MemberService memberService;
    @Mock private PasswordEncryptor passwordEncryptor;
    @Mock private JwtProvider jwtProvider;
    @Mock private CacheService cacheService;

    // 테스트용 상수
    private final Long REMAINING_DUE = 1000L * 60 * 60 * 24; // 1일 (설정값 모킹)
    private final String CACHE_NAME = "refresh_tokens";

    @BeforeEach
    void setUp() {
        // 생성자 주입을 직접 호출하여 @Value 값을 넣어줍니다.
        authService = new AuthService(
                memberService,
                passwordEncryptor,
                jwtProvider,
                cacheService,
                REMAINING_DUE
        );
    }

    @Nested
    @DisplayName("에이전트 키 발급 테스트")
    class IssueKeyTest {
        @Test
        @DisplayName("존재하지 않는 멤버가 Client Key 요청 시 예외가 발생한다")
        void fail_client_key_member_not_found() {
            // given
            given(memberService.existsById(any())).willReturn(false);

            // when & then
            assertThatThrownBy(() -> authService.issueClientKeyByMemberId(999L))
                    .isInstanceOf(BaseException.class)
                    .extracting("code").isEqualTo(ENTITY_NOT_FOUND.getCode());
        }

        @Test
        @DisplayName("존재하지 않는 멤버가 Worker Key 요청 시 예외가 발생한다")
        void fail_worker_key_member_not_found() {
            // given
            given(memberService.existsById(any())).willReturn(false);

            // when & then
            // 코드상 Worker는 ResponseStatusException을 던짐
            assertThatThrownBy(() -> authService.issueWorkerKeyByMemberId(999L))
                    .isInstanceOf(BaseException.class);
        }
    }

    @Nested
    @DisplayName("회원가입/로그인 테스트")
    class SignupSigninTest {

        @Test
        @DisplayName("회원가입 시 비밀번호가 암호화되고 토큰이 캐시에 저장된다")
        void signup_success() {
            // given
            NormalSignupRequest request = new NormalSignupRequest("test@test.com", "nick", "pw");
            Member member = Member.builder()
                    .email("test@test.com")
                    .password("pw")
                    .build();
            ReflectionTestUtils.setField(member, "id", 1L);

            given(memberService.existsByEmail(any())).willReturn(false);
            given(passwordEncryptor.encrypt(any())).willReturn("encodedPw");
            given(memberService.createMember(any())).willReturn(member);
            given(jwtProvider.createAccessToken(any())).willReturn("access");
            given(jwtProvider.createRefreshToken(any())).willReturn("refresh");

            // when
            TokenResponse response = authService.signup(request);

            // then
            assertThat(response.getAccessToken()).isEqualTo("access");
            verify(passwordEncryptor).encrypt("pw"); // 암호화 수행 확인
            verify(cacheService).put(eq(CACHE_NAME), eq("1"), eq("refresh")); // 캐시 저장 확인
        }

        @Test
        @DisplayName("로그인 시 비밀번호가 틀리면 예외가 발생한다")
        void signin_fail_wrong_password() {
            // given
            NormalSigninRequest request = new NormalSigninRequest("test@test.com", "wrongPw");
            Member member = Member.builder()
                    .email("test@test.com")
                    .password("encodedPw").build();
            ReflectionTestUtils.setField(member, "id", 1L);

            given(memberService.findByEmail(any())).willReturn(member);
            given(passwordEncryptor.match(any(), any())).willReturn(false); // 불일치

            // when & then
            assertThatThrownBy(() -> authService.signin(request))
                    .isInstanceOf(BaseException.class)
                    .extracting("code").isEqualTo(INVALID_PASSWORD.getCode());
        }
    }

    @Nested
    @DisplayName("토큰 재발급(Reissue) 테스트 - 핵심")
    class ReissueTest {

        private final String OLD_REFRESH = "old-refresh-token";
        private final String NEW_ACCESS = "new-access-token";
        private final String NEW_REFRESH = "new-refresh-token";
        private final Long MEMBER_ID = 1L;

        @Test
        @DisplayName("[Rotation] 만료 시간이 임박하면(remainingDue 미만) 새 Refresh Token을 발급하고 캐시를 갱신한다")
        void reissue_rotation_success() {
            // given
            ReissueRequest request = new ReissueRequest(OLD_REFRESH);
            Member member = Member.builder()
                    .build();
            ReflectionTestUtils.setField(member, "id", MEMBER_ID);

            // 1. 토큰 검증 통과
            given(jwtProvider.validateToken(OLD_REFRESH)).willReturn(true);
            given(jwtProvider.getMemberId(OLD_REFRESH)).willReturn(MEMBER_ID);

            // 2. 캐시 확인 (정상)
            given(cacheService.get(eq(CACHE_NAME), anyString(), eq(String.class)))
                    .willReturn(Optional.of(OLD_REFRESH));

            // 3. 새 Access Token 생성
            given(jwtProvider.createAccessToken(MEMBER_ID)).willReturn(NEW_ACCESS);

            // 4. [핵심] 남은 시간이 설정값보다 적음 (1초 남음) -> Rotation 트리거
            given(jwtProvider.getRemainingTime(OLD_REFRESH)).willReturn(1000L);
            // 1000L < REMAINING_DUE(하루) 이므로 true

            given(jwtProvider.createRefreshToken(MEMBER_ID)).willReturn(NEW_REFRESH);

            // when
            TokenResponse response = authService.reissue(request);

            // then
            assertThat(response.getAccessToken()).isEqualTo(NEW_ACCESS);
            assertThat(response.getRefreshToken()).isEqualTo(NEW_REFRESH); // 새 토큰인지 확인

            // 캐시에 "새로운" 토큰이 저장되었는지 검증
            verify(cacheService).put(CACHE_NAME, String.valueOf(MEMBER_ID), NEW_REFRESH);
        }

        @Test
        @DisplayName("[No Rotation] 만료 시간이 충분하면(remainingDue 이상) 기존 Refresh Token을 유지하고 캐시를 갱신하지 않는다")
        void reissue_no_rotation_success() {
            // given
            ReissueRequest request = new ReissueRequest(OLD_REFRESH);

            given(jwtProvider.validateToken(OLD_REFRESH)).willReturn(true);
            given(jwtProvider.getMemberId(OLD_REFRESH)).willReturn(MEMBER_ID);
            given(cacheService.get(eq(CACHE_NAME), anyString(), eq(String.class)))
                    .willReturn(Optional.of(OLD_REFRESH));
            given(jwtProvider.createAccessToken(MEMBER_ID)).willReturn(NEW_ACCESS);

            // 4. [핵심] 남은 시간이 충분함 (2일 남음)
            long enoughTime = REMAINING_DUE + 10000L;
            given(jwtProvider.getRemainingTime(OLD_REFRESH)).willReturn(enoughTime);

            // when
            TokenResponse response = authService.reissue(request);

            // then
            assertThat(response.getAccessToken()).isEqualTo(NEW_ACCESS);
            assertThat(response.getRefreshToken()).isEqualTo(OLD_REFRESH); // "기존" 토큰인지 확인

            // 새 토큰 생성 메서드가 호출되지 않았는지 확인
            verify(jwtProvider, never()).createRefreshToken(any());
            // 캐시 저장 메서드가 호출되지 않았는지 확인 (성능 최적화)
            verify(cacheService, never()).put(any(), any(), any());
        }

        @Test
        @DisplayName("저장소에 토큰이 없으면(로그아웃/만료) 예외가 발생한다")
        void reissue_fail_not_in_cache() {
            // given
            given(jwtProvider.validateToken(OLD_REFRESH)).willReturn(true);
            given(jwtProvider.getMemberId(OLD_REFRESH)).willReturn(MEMBER_ID);

            // 캐시에 없음 (Optional.empty)
            given(cacheService.get(eq(CACHE_NAME), anyString(), eq(String.class)))
                    .willReturn(Optional.empty());

            // when & then
            assertThatThrownBy(() -> authService.reissue(new ReissueRequest(OLD_REFRESH)))
                    .isInstanceOf(BaseException.class)
                    .extracting("code").isEqualTo(EXPIRED_REFRESH_TOKEN.getCode());
        }
    }
}