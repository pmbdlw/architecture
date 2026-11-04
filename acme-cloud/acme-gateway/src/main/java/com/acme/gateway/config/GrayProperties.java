package com.acme.gateway.config;

import java.util.ArrayList;
import java.util.List;
import org.springframework.boot.context.properties.ConfigurationProperties;

/**
 * 灰度规则，放在 Nacos 的 acme-gateway.yaml 中，修改后实时生效。
 * <p>
 * 必须是可变的 JavaBean：Nacos 配置变更后 Spring Cloud 通过 ConfigurationPropertiesRebinder 重新绑定，
 * 而 record / 构造器绑定的配置类不会被重新绑定——那样规则只在启动时生效，灰度开关形同虚设。
 */
@ConfigurationProperties(prefix = "acme.gateway.gray")
public class GrayProperties {

    /** 是否开启灰度。 */
    private boolean enabled;

    /** 灰度版本标签，与灰度 Deployment 的 GRAY_TAG 环境变量一致，如 v2。 */
    private String tag;

    /** 白名单用户（JWT uid）。 */
    private List<String> uids = new ArrayList<>();

    /** 按用户 ID 哈希放量的百分比 0-100。 */
    private int percentage;

    /** 是否信任客户端自带的 X-Gray-Tag（仅 test/staging 打开，便于测试同学指定版本）。 */
    private boolean allowClientHeader;

    public GrayProperties() {
    }

    public GrayProperties(boolean enabled, String tag, List<String> uids, int percentage, boolean allowClientHeader) {
        this.enabled = enabled;
        this.tag = tag;
        this.uids = new ArrayList<>(uids);
        this.percentage = percentage;
        this.allowClientHeader = allowClientHeader;
    }

    public boolean isEnabled() {
        return enabled;
    }

    public void setEnabled(boolean enabled) {
        this.enabled = enabled;
    }

    public String getTag() {
        return tag;
    }

    public void setTag(String tag) {
        this.tag = tag;
    }

    public List<String> getUids() {
        return uids;
    }

    public void setUids(List<String> uids) {
        this.uids = uids;
    }

    public int getPercentage() {
        return percentage;
    }

    public void setPercentage(int percentage) {
        this.percentage = percentage;
    }

    public boolean isAllowClientHeader() {
        return allowClientHeader;
    }

    public void setAllowClientHeader(boolean allowClientHeader) {
        this.allowClientHeader = allowClientHeader;
    }
}
