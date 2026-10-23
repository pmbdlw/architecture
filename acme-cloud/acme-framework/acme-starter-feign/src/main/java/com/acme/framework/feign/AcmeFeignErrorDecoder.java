package com.acme.framework.feign;

import java.io.IOException;
import java.io.InputStream;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.acme.framework.core.exception.BizException;
import com.acme.framework.core.exception.RemoteErrorCode;
import com.acme.framework.core.exception.RemoteServiceException;

import feign.Response;
import feign.codec.ErrorDecoder;
import tools.jackson.databind.JsonNode;
import tools.jackson.databind.json.JsonMapper;

/**
 * 把下游的错误响应还原为平台异常：
 * <ul>
 *   <li>4xx 且响应体是 Result：还原为 {@link BizException}（不计入熔断统计）；</li>
 *   <li>其它情况：{@link RemoteServiceException}（计入熔断统计，对外表现为 C0001/C0503）。</li>
 * </ul>
 */
public class AcmeFeignErrorDecoder implements ErrorDecoder {

    private static final Logger log = LoggerFactory.getLogger(AcmeFeignErrorDecoder.class);

    private final JsonMapper jsonMapper;

    public AcmeFeignErrorDecoder(JsonMapper jsonMapper) {
        this.jsonMapper = jsonMapper;
    }

    @Override
    public Exception decode(String methodKey, Response response) {
        String service = response.request().requestTemplate().feignTarget().name();
        int status = response.status();
        JsonNode body = readBody(response);
        if (status >= 400 && status < 500 && body != null && body.hasNonNull("code")) {
            String message = body.path("message").asString("");
            return new BizException(new RemoteErrorCode(body.get("code").asString(), message, status), message,
                    null);
        }
        log.warn("调用 {} 失败: {} status={}", service, methodKey, status);
        return new RemoteServiceException(service, status, response.reason());
    }

    private JsonNode readBody(Response response) {
        if (response.body() == null) {
            return null;
        }
        try (InputStream in = response.body().asInputStream()) {
            return jsonMapper.readTree(in);
        } catch (IOException | RuntimeException ex) {
            return null;
        }
    }
}
