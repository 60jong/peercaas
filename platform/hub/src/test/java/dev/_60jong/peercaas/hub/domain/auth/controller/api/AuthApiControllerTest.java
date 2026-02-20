package dev._60jong.peercaas.hub.domain.auth.controller.api;

import dev._60jong.peercaas.hub.domain.auth.controller.api.request.ReissueRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.request.NormalSigninRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.request.NormalSignupRequest;
import dev._60jong.peercaas.hub.domain.auth.controller.api.response.GetKeyResponse;
import dev._60jong.peercaas.hub.domain.auth.controller.api.response.TokenResponse;
import dev._60jong.peercaas.hub.domain.auth.service.AuthService;
import dev._60jong.peercaas.hub.support.RestDocsSupport;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.restdocs.payload.JsonFieldType;

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
class AuthApiControllerDocsTest extends RestDocsSupport {

    @MockBean
    private AuthService authService;

    @Test
    @DisplayName("Client Agent 키 발급 API")
    void getClientKey() throws Exception {
        // given
        Long memberId = 1L;
        GetKeyResponse response = new GetKeyResponse("client-agent-key-1234");

        given(authService.issueClientKeyByMemberId(memberId)).willReturn(response);

        // when & then
        mockMvc.perform(get("/api/v1/auth/agent/client/key")
                        .param("memberId", String.valueOf(memberId))
                        .contentType(MediaType.APPLICATION_JSON))
                .andExpect(status().isOk())
                .andDo(document("auth-client-key",
                        queryParameters(
                                parameterWithName("memberId").description("회원 ID")
                        ),
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
        GetKeyResponse response = new GetKeyResponse("worker-agent-key-5678");

        given(authService.issueWorkerKeyByMemberId(memberId)).willReturn(response);

        // when & then
        mockMvc.perform(get("/api/v1/auth/agent/worker/key")
                        .param("memberId", String.valueOf(memberId))
                        .contentType(MediaType.APPLICATION_JSON))
                .andExpect(status().isOk())
                .andDo(document("auth-worker-key",
                        queryParameters(
                                parameterWithName("memberId").description("회원 ID")
                        ),
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

        TokenResponse response = new TokenResponse("access-token", "refresh-token");

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
                                fieldWithPath("data.refreshToken").type(JsonFieldType.STRING).description("리프레시 토큰")
                        )
                ));
    }

    @Test
    @DisplayName("로그인 API")
    void signin() throws Exception {
        // given
        NormalSigninRequest request = new NormalSigninRequest("test@test.com", "password123");
        TokenResponse response = new TokenResponse("access-token", "refresh-token");

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
                                fieldWithPath("data.refreshToken").type(JsonFieldType.STRING).description("리프레시 토큰")
                        )
                ));
    }

    @Test
    @DisplayName("토큰 재발급 API")
    void reissue() throws Exception {
        // given
        ReissueRequest request = new ReissueRequest("old-refresh-token");
        TokenResponse response = new TokenResponse("new-access-token", "new-refresh-token");

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
                                fieldWithPath("data.refreshToken").type(JsonFieldType.STRING).description("새로운 리프레시 토큰 (Rotation 시)")
                        )
                ));
    }
}