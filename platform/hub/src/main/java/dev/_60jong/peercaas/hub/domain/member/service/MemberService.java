package dev._60jong.peercaas.hub.domain.member.service;

import dev._60jong.peercaas.hub.domain.auth.util.PasswordEncryptor;
import dev._60jong.peercaas.hub.domain.member.controller.api.request.CreateMemberRequest;
import dev._60jong.peercaas.hub.domain.member.controller.api.response.CreateMemberResponse;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import dev._60jong.peercaas.hub.domain.member.model.vo.MemberParam;
import dev._60jong.peercaas.hub.domain.member.repository.MemberRepository;
import dev._60jong.peercaas.hub.global.exception.BaseException;
import dev._60jong.peercaas.hub.global.exception.member.MemberExceptionCode;
import jakarta.transaction.Transactional;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Service;
import org.springframework.web.server.ResponseStatusException;

import static dev._60jong.peercaas.hub.global.exception.member.MemberExceptionCode.ENTITY_NOT_FOUND;

@RequiredArgsConstructor
@Service
public class MemberService {

    private final MemberRepository memberRepository;

    private final PasswordEncryptor passwordEncryptor;

    public boolean existsById(Long memberId) {
        return memberRepository.existsById(memberId);
    }

    public Member findById(Long memberId) {
        return memberRepository.findById(memberId)
                .orElseThrow(() -> new ResponseStatusException(HttpStatus.NOT_FOUND));
    }

    @Transactional
    public Member createMember(MemberParam param) {
        Member member = Member.builder()
                .nickname(param.getNickname())
                .email(param.getEmail())
                .password(param.getPassword())
                .build();

        memberRepository.save(member);
        return member;
    }

    @Transactional
    public void updateMember(Long memberId, String nickname) {
        Member member = findById(memberId);
        member.updateProfile(nickname);
    }

    @Transactional
    public void resetClientKey(Long memberId) {
        Member member = findById(memberId);
        member.resetClientKey();
    }

    public Member findByEmail(String email) {
        return memberRepository.findByEmail(email)
                .orElseThrow(() -> new BaseException(ENTITY_NOT_FOUND, "존재하지 않는 Email입니다."));
    }

    public boolean existsByEmail(String email) {
        return memberRepository.existsByEmail(email);
    }

}
