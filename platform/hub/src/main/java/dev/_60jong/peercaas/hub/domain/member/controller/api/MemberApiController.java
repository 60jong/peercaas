package dev._60jong.peercaas.hub.domain.member.controller.api;

import dev._60jong.peercaas.common.web.api.response.ApiResponse;
import dev._60jong.peercaas.hub.domain.member.controller.api.request.CreateMemberRequest;
import dev._60jong.peercaas.hub.domain.member.controller.api.response.CreateMemberResponse;
import dev._60jong.peercaas.hub.domain.member.service.MemberService;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RequiredArgsConstructor
@RestController
@RequestMapping("/api/v1/member")
public class MemberApiController {

    private final MemberService memberService;


}
