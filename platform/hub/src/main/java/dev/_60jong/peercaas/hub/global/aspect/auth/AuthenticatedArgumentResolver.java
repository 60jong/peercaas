package dev._60jong.peercaas.hub.global.aspect.auth;

import dev._60jong.peercaas.hub.domain.auth.util.JwtProvider;
import dev._60jong.peercaas.hub.global.exception.BaseException;
import lombok.RequiredArgsConstructor;
import org.springframework.core.MethodParameter;
import org.springframework.stereotype.Component;
import org.springframework.util.StringUtils;
import org.springframework.web.bind.support.WebDataBinderFactory;
import org.springframework.web.context.request.NativeWebRequest;
import org.springframework.web.method.support.HandlerMethodArgumentResolver;
import org.springframework.web.method.support.ModelAndViewContainer;

import static dev._60jong.peercaas.hub.global.exception.auth.AuthExceptionCode.INVALID_TOKEN;
import static dev._60jong.peercaas.hub.global.exception.auth.AuthExceptionCode.TOKEN_REQUIRED;

@RequiredArgsConstructor
@Component
public class AuthenticatedArgumentResolver implements HandlerMethodArgumentResolver {

    private static final String AUTHORIZATION_HEADER = "Authorization";
    private static final String BEARER_PREFIX = "Bearer ";

    private final JwtProvider jwtProvider;

    @Override
    public boolean supportsParameter(MethodParameter parameter) {
        return parameter.hasParameterAnnotation(Authenticated.class)
                && parameter.getParameterType().equals(Long.class);
    }

    @Override
    public Object resolveArgument(
            MethodParameter parameter,
            ModelAndViewContainer mavContainer,
            NativeWebRequest webRequest,
            WebDataBinderFactory binderFactory
    ) {
        String token = extractToken(webRequest);
        Authenticated annotation = parameter.getParameterAnnotation(Authenticated.class);
        boolean required = annotation != null && annotation.required();

        if (token == null) {
            if (required) {
                throw new BaseException(TOKEN_REQUIRED, "인증이 필요합니다.");
            }
            return null;
        }

        // 토큰 유효성 검증 실패 시에도 required 여부에 따라 분기
        try {
            return jwtProvider.getMemberId(token);
        } catch (Exception e) {
            if (required) {
                throw new BaseException(INVALID_TOKEN, "유효하지 않은 토큰입니다.");
            }
            return null;
        }
    }

    private String extractToken(NativeWebRequest webRequest) {
        String header = webRequest.getHeader(AUTHORIZATION_HEADER);

        if (StringUtils.hasText(header) && header.startsWith(BEARER_PREFIX)) {
            return header.substring(BEARER_PREFIX.length());
        }
        return null;
    }
}
