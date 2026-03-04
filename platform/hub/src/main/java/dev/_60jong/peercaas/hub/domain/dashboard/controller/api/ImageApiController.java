package dev._60jong.peercaas.hub.domain.dashboard.controller.api;

import dev._60jong.peercaas.common.web.api.response.ApiResponse;
import lombok.Getter;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.client.HttpClientErrorException;
import org.springframework.web.client.RestTemplate;

@RestController
@RequestMapping("/api/v1/images")
public class ImageApiController {

    private final RestTemplate restTemplate = new RestTemplate();

    /**
     * Docker Hub에서 이미지/태그 존재 여부 검증
     * repo 없으면 공식 이미지(library) 기준으로 확인
     */
    @GetMapping("/validate")
    public ApiResponse<ImageValidateResponse> validate(
            @RequestParam(required = false) String repo,
            @RequestParam String image,
            @RequestParam(defaultValue = "latest") String tag
    ) {
        String namespace = (repo != null && !repo.isBlank()) ? repo : "library";
        String url = String.format(
                "https://hub.docker.com/v2/repositories/%s/%s/tags/%s/",
                namespace, image, tag
        );

        try {
            restTemplate.getForEntity(url, String.class);
            String fullImage = "library".equals(namespace)
                    ? image + ":" + tag
                    : namespace + "/" + image + ":" + tag;
            return ApiResponse.success(new ImageValidateResponse(true, fullImage, "library".equals(namespace)));
        } catch (HttpClientErrorException.NotFound e) {
            return ApiResponse.success(new ImageValidateResponse(false, null, false));
        } catch (Exception e) {
            return ApiResponse.success(new ImageValidateResponse(false, null, false));
        }
    }

    /**
     * Docker Hub 리포지토리 검색
     */
    @GetMapping("/search")
    public ApiResponse<Object> search(@RequestParam String q) {
        String url = String.format("https://hub.docker.com/v2/search/repositories/?query=%s", q);
        try {
            // Docker Hub 응답을 그대로 중계
            Object response = restTemplate.getForObject(url, Object.class);
            return ApiResponse.success(response);
        } catch (Exception e) {
            return ApiResponse.success(null);
        }
    }

    @Getter
    public static class ImageValidateResponse {
        private final boolean exists;
        private final String fullImage;
        private final boolean official;

        public ImageValidateResponse(boolean exists, String fullImage, boolean official) {
            this.exists = exists;
            this.fullImage = fullImage;
            this.official = official;
        }
    }
}
