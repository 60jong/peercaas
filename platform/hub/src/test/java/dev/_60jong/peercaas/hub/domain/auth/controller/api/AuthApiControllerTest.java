package dev._60jong.peercaas.hub.domain.auth.controller.api;

import dev._60jong.peercaas.hub.domain.auth.controller.api.request.ReissueRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.request.NormalSigninRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.request.NormalSignupRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.response.GetKeyResponse;
import dev._60jong.peercaas.hub.domain.auth.controller.api.response.TokenResponse;
import dev._60jong.peercaas.hub.domain.auth.service.AuthService;
import dev._60jong.peercaas.hub.domain.auth.util.JwtProvider;
import dev._60jong.peercaas.hub.domain.member.model.entity.Member;
import dev._60jong.peercaas.hub.domain.member.service.MemberService;
import dev._60jong.peercaas.hub.global.aspect.auth.AuthenticatedArgumentResolver;
import dev._60jong.peercaas.hub.support.RestDocsSupport;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.restdocs.payload.JsonFieldType;
import org.springframework.test.util.ReflectionTestUtils;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.BDDMockito.given;
import static org.springframework.restdocs.mockmvc.MockMvcRestDocumentation.document;
import static org.springframework.restdocs.mockmvc.RestDocumentationRequestBuilders.get;
import static org.springframework.restdocs.mockmvc.RestDocumentationRequestBuilders.post;
import static org.springframework.restdocs.payload.PayloadDocumentation.*;
import static org.springframework.restdocs.request.RequestDocumentation.parameterWithName;
import static org.springframework.restdocs.request.RequestDocumentation.queryParameters;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(AuthApiController.class)
class AuthApiControllerTest extends RestDocsSupport {

    @MockBean
    private AuthService authService;

    @MockBean
    private MemberService memberService;

    @MockBean
    private AuthenticatedArgumentResolver authenticatedArgumentResolver;

    @MockBean
    private JwtProvider jwtProvider;

    @Test
    @DisplayName("Client Agent 키 발급 API")
    void getClientKey() throws Exception {
        // given
        Long memberId = 1L;
        String clientKey = "client-agent-key-1234";
        Member member = Member.builder()
                .nickname("tester")
                .build();
        ReflectionTestUtils.setField(member, "clientKey", clientKey);

        given(authenticatedArgumentResolver.supportsParameter(any())).willReturn(true);
        given(authenticatedArgumentResolver.resolveArgument(any(), any(), any(), any())).willReturn(memberId);
        given(memberService.findById(memberId)).willReturn(member);

        // when & then
        mockMvc.perform(get("/api/v1/auth/agent/client/key")
                        .contentType(MediaType.APPLICATION_JSON))
                .andExpect(status().isOk())
                .andDo(document("auth-client-key",
                        responseFields(
                                fieldWithPath("timestamp").type(JsonFieldType.STRING).description("응답 시간"),
                                fieldWithPath("code").type(JsonFieldType.NUMBER).description("응답 코드"),
                                fieldWithPath("message").type(JsonFieldType.STRING).description("응답 메시지"),
                                fieldWithPath("data.key").type(JsonFieldType.STRING).description("발급된 클라이언트 키")
                        )
                ));
    }

    @Test
    @DisplayName("Worker Agent 키 발급 API")
    void getWorkerKey() throws Exception {
        // given
        Long memberId = 1L;
        String workerKey = "worker-agent-key-5678";
        Member member = Member.builder()
                .nickname("tester")
                .build();
        ReflectionTestUtils.setField(member, "workerKey", workerKey);

        given(authenticatedArgumentResolver.supportsParameter(any())).willReturn(true);
        given(authenticatedArgumentResolver.resolveArgument(any(), any(), any(), any())).willReturn(memberId);
        given(memberService.findById(memberId)).willReturn(member);

        // when & then
        mockMvc.perform(get("/api/v1/auth/agent/worker/key")
                        .contentType(MediaType.APPLICATION_JSON))
                .andExpect(status().isOk())
                .andDo(document("auth-worker-key",
                        responseFields(
                                fieldWithPath("timestamp").type(JsonFieldType.STRING).description("응답 시간"),
                                fieldWithPath("code").type(JsonFieldType.NUMBER).description("응답 코드"),
                                fieldWithPath("message").type(JsonFieldType.STRING).description("응답 메시지"),
                                fieldWithPath("data.key").type(JsonFieldType.STRING).description("발급된 워커 키")
                        )
                ));
    }

    @Test
    @DisplayName("회원가입 API")
    void signup() throws Exception {
        // given
        NormalSignupRequest request = new NormalSignupRequest("tester", "test@test.com", "password123");

        TokenResponse response = new TokenResponse("access-token", "refresh-token", "tester");

        given(authService.signup(any(NormalSignupRequest.class))).willReturn(response);

        // when & then
        mockMvc.perform(post("/api/v1/auth/signup")
                        .content(objectMapper.writeValueAsString(request))
                        .contentType(MediaType.APPLICATION_JSON))
                .andExpect(status().isOk())
                .andDo(document("auth-signup",
                        requestFields(
                                fieldWithPath("email").type(JsonFieldType.STRING).description("이메일"),
                                fieldWithPath("password").type(JsonFieldType.STRING).description("비밀번호"),
                                fieldWithPath("nickname").type(JsonFieldType.STRING).description("닉네임")
                        ),
                        responseFields(
                                fieldWithPath("timestamp").type(JsonFieldType.STRING).description("응답 시간"),
                                fieldWithPath("code").type(JsonFieldType.NUMBER).description("응답 코드"),
                                fieldWithPath("message").type(JsonFieldType.STRING).description("응답 메시지"),
                                fieldWithPath("data.accessToken").type(JsonFieldType.STRING).description("액세스 토큰"),
                                fieldWithPath("data.refreshToken").type(JsonFieldType.STRING).description("리프레시 토큰"),
                                fieldWithPath("data.nickname").type(JsonFieldType.STRING).description("닉네임")
                        )
                ));
    }

    @Test
    @DisplayName("로그인 API")
    void signin() throws Exception {
        // given
        NormalSigninRequest request = new NormalSigninRequest("test@test.com", "password123");
        TokenResponse response = new TokenResponse("access-token", "refresh-token", "tester");

        given(authService.signin(any(NormalSigninRequest.class))).willReturn(response);

        // when & then
        mockMvc.perform(post("/api/v1/auth/signin")
                        .content(objectMapper.writeValueAsString(request))
                        .contentType(MediaType.APPLICATION_JSON))
                .andExpect(status().isOk())
                .andDo(document("auth-signin",
                        requestFields(
                                fieldWithPath("email").type(JsonFieldType.STRING).description("이메일"),
                                fieldWithPath("password").type(JsonFieldType.STRING).description("비밀번호")
                        ),
                        responseFields(
                                fieldWithPath("timestamp").type(JsonFieldType.STRING).description("응답 시간"),
                                fieldWithPath("code").type(JsonFieldType.NUMBER).description("응답 코드"),
                                fieldWithPath("message").type(JsonFieldType.STRING).description("응답 메시지"),
                                fieldWithPath("data.accessToken").type(JsonFieldType.STRING).description("액세스 토큰"),
                                fieldWithPath("data.refreshToken").type(JsonFieldType.STRING).description("리프레시 토큰"),
                                fieldWithPath("data.nickname").type(JsonFieldType.STRING).description("닉네임")
                        )
                ));
    }

    @Test
    @DisplayName("토큰 재발급 API")
    void reissue() throws Exception {
        // given
        ReissueRequest request = new ReissueRequest("old-refresh-token");
        TokenResponse response = new TokenResponse("new-access-token", "new-refresh-token", "tester");

        given(authService.reissue(any(ReissueRequest.class))).willReturn(response);

        // when & then
        mockMvc.perform(post("/api/v1/auth/reissue")
                        .content(objectMapper.writeValueAsString(request))
                        .contentType(MediaType.APPLICATION_JSON))
                .andExpect(status().isOk())
                .andDo(document("auth-reissue",
                        requestFields(
                                fieldWithPath("refreshToken").type(JsonFieldType.STRING).description("기존 리프레시 토큰")
                        ),
                        responseFields(
                                fieldWithPath("timestamp").type(JsonFieldType.STRING).description("응답 시간"),
                                fieldWithPath("code").type(JsonFieldType.NUMBER).description("응답 코드"),
                                fieldWithPath("message").type(JsonFieldType.STRING).description("응답 메시지"),
                                fieldWithPath("data.accessToken").type(JsonFieldType.STRING).description("새로운 액세스 토큰"),
                                fieldWithPath("data.refreshToken").type(JsonFieldType.STRING).description("새로운 리프레시 토큰 (Rotation 시)"),
                                fieldWithPath("data.nickname").type(JsonFieldType.STRING).description("닉네임")
                        )
                ));
    }
}