package dev._60jong.peercaas.hub.global.aspect.auth;

import dev._60jong.peercaas.hub.domain.auth.util.JwtProvider;
import dev._60jong.peercaas.hub.global.exception.BaseException;
import dev._60jong.peercaas.hub.global.exception.auth.AuthExceptionCode;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.core.MethodParameter;
import org.springframework.web.context.request.NativeWebRequest;

import java.lang.annotation.Annotation;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.BDDMockito.given;

@ExtendWith(MockitoExtension.class)
class AuthenticatedArgumentResolverTest {

    @InjectMocks
    private AuthenticatedArgumentResolver resolver;

    @Mock
    private JwtProvider jwtProvider;

    @Mock
    private NativeWebRequest webRequest;

    @Mock
    private MethodParameter parameter;

    private static final Long MEMBER_ID = 1L;
    private static final String VALID_TOKEN = "valid.jwt.token";

    // required = true인 @Authenticated 어노테이션 생성
    private Authenticated authenticatedRequired() {
        return new Authenticated() {
            @Override
            public Class<? extends Annotation> annotationType() {
                return Authenticated.class;
            }

            @Override
            public boolean required() {
                return true;
            }
        };
    }

    // required = false인 @Authenticated 어노테이션 생성
    private Authenticated authenticatedOptional() {
        return new Authenticated() {
            @Override
            public Class<? extends Annotation> annotationType() {
                return Authenticated.class;
            }

            @Override
            public boolean required() {
                return false;
            }
        };
    }

    @Nested
    @DisplayName("supportsParameter")
    class SupportsParameter {

        @Test
        @DisplayName("@Authenticated가 붙은 Long 파라미터는 지원한다")
        void supports_authenticated_long_parameter() {
            given(parameter.hasParameterAnnotation(Authenticated.class)).willReturn(true);
            given(parameter.getParameterType()).willReturn((Class) Long.class);

            assertThat(resolver.supportsParameter(parameter)).isTrue();
        }

        @Test
        @DisplayName("@Authenticated가 없으면 지원하지 않는다")
        void does_not_support_without_annotation() {
            given(parameter.hasParameterAnnotation(Authenticated.class)).willReturn(false);

            assertThat(resolver.supportsParameter(parameter)).isFalse();
        }

        @Test
        @DisplayName("@Authenticated가 있어도 Long 타입이 아니면 지원하지 않는다")
        void does_not_support_non_long_type() {
            given(parameter.hasParameterAnnotation(Authenticated.class)).willReturn(true);
            given(parameter.getParameterType()).willReturn((Class) String.class);

            assertThat(resolver.supportsParameter(parameter)).isFalse();
        }
    }

    @Nested
    @DisplayName("resolveArgument - 토큰이 유효한 경우")
    class ValidToken {

        @BeforeEach
        void setUp() {
            given(webRequest.getHeader("Authorization")).willReturn("Bearer " + VALID_TOKEN);
            given(jwtProvider.getMemberId(VALID_TOKEN)).willReturn(MEMBER_ID);
        }

        @Test
        @DisplayName("required=true일 때 memberId를 반환한다")
        void returns_memberId_when_required() {
            given(parameter.getParameterAnnotation(Authenticated.class))
                    .willReturn(authenticatedRequired());

            Object result = resolver.resolveArgument(parameter, null, webRequest, null);

            assertThat(result).isEqualTo(MEMBER_ID);
        }

        @Test
        @DisplayName("required=false일 때도 memberId를 반환한다")
        void returns_memberId_when_optional() {
            given(parameter.getParameterAnnotation(Authenticated.class))
                    .willReturn(authenticatedOptional());

            Object result = resolver.resolveArgument(parameter, null, webRequest, null);

            assertThat(result).isEqualTo(MEMBER_ID);
        }
    }

    @Nested
    @DisplayName("resolveArgument - 토큰이 없는 경우")
    class NoToken {

        @BeforeEach
        void setUp() {
            given(webRequest.getHeader("Authorization")).willReturn(null);
        }

        @Test
        @DisplayName("required=true일 때 TOKEN_REQUIRED 예외를 던진다")
        void throws_exception_when_required() {
            given(parameter.getParameterAnnotation(Authenticated.class))
                    .willReturn(authenticatedRequired());

            assertThatThrownBy(() -> resolver.resolveArgument(parameter, null, webRequest, null))
                    .isInstanceOf(BaseException.class)
                    .satisfies(ex -> assertThat(((BaseException) ex).getCode())
                            .isEqualTo(AuthExceptionCode.TOKEN_REQUIRED.getCode()));
        }

        @Test
        @DisplayName("required=false일 때 null을 반환한다")
        void returns_null_when_optional() {
            given(parameter.getParameterAnnotation(Authenticated.class))
                    .willReturn(authenticatedOptional());

            Object result = resolver.resolveArgument(parameter, null, webRequest, null);

            assertThat(result).isNull();
        }
    }

    @Nested
    @DisplayName("resolveArgument - 토큰이 유효하지 않은 경우")
    class InvalidToken {

        @BeforeEach
        void setUp() {
            String invalidToken = "invalid.jwt.token";
            given(webRequest.getHeader("Authorization")).willReturn("Bearer " + invalidToken);
            given(jwtProvider.getMemberId(invalidToken)).willThrow(new RuntimeException("토큰 파싱 실패"));
        }

        @Test
        @DisplayName("required=true일 때 INVALID_TOKEN 예외를 던진다")
        void throws_exception_when_required() {
            given(parameter.getParameterAnnotation(Authenticated.class))
                    .willReturn(authenticatedRequired());

            assertThatThrownBy(() -> resolver.resolveArgument(parameter, null, webRequest, null))
                    .isInstanceOf(BaseException.class)
                    .satisfies(ex -> assertThat(((BaseException) ex).getCode())
                            .isEqualTo(AuthExceptionCode.INVALID_TOKEN.getCode()));
        }

        @Test
        @DisplayName("required=false일 때 null을 반환한다")
        void returns_null_when_optional() {
            given(parameter.getParameterAnnotation(Authenticated.class))
                    .willReturn(authenticatedOptional());

            Object result = resolver.resolveArgument(parameter, null, webRequest, null);

            assertThat(result).isNull();
        }
    }

    @Nested
    @DisplayName("resolveArgument - Authorization 헤더 형식")
    class HeaderFormat {

        @Test
        @DisplayName("Bearer 접두사 없이 토큰만 있으면 토큰 없음으로 처리한다")
        void no_bearer_prefix_treated_as_no_token() {
            given(webRequest.getHeader("Authorization")).willReturn(VALID_TOKEN);
            given(parameter.getParameterAnnotation(Authenticated.class))
                    .willReturn(authenticatedRequired());

            assertThatThrownBy(() -> resolver.resolveArgument(parameter, null, webRequest, null))
                    .isInstanceOf(BaseException.class)
                    .satisfies(ex -> assertThat(((BaseException) ex).getCode())
                            .isEqualTo(AuthExceptionCode.TOKEN_REQUIRED.getCode()));
        }

        @Test
        @DisplayName("빈 문자열 헤더는 토큰 없음으로 처리한다")
        void empty_header_treated_as_no_token() {
            given(webRequest.getHeader("Authorization")).willReturn("");
            given(parameter.getParameterAnnotation(Authenticated.class))
                    .willReturn(authenticatedOptional());

            Object result = resolver.resolveArgument(parameter, null, webRequest, null);

            assertThat(result).isNull();
        }
    }
}