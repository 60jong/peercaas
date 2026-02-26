package dev._60jong.peercaas.hub.domain.member.controller.api;

import dev._60jong.peercaas.common.web.api.response.ApiResponse;
import dev._60jong.peercaas.hub.domain.member.controller.api.request.CreateMemberRequest;
import dev._60jong.peercaas.hub.domain.member.controller.api.response.CreateMemberResponse;
import dev._60jong.peercaas.hub.domain.member.controller.api.response.MemberProfileResponse;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import dev._60jong.peercaas.hub.domain.member.service.MemberService;
import dev._60jong.peercaas.hub.global.aspect.auth.Authenticated;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.*;

@RequiredArgsConstructor
@RestController
@RequestMapping("/api/v1/member")
public class MemberApiController {

    private final MemberService memberService;

    @GetMapping("/me")
    public ApiResponse<MemberProfileResponse> getMe(@Authenticated Long memberId) {
        Member member = memberService.findById(memberId);
        return ApiResponse.success(new MemberProfileResponse(
                member.getId(),
                member.getEmail(),
                member.getNickname()
        ));
    }

}
