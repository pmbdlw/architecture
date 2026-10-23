package com.acme.gateway.filter;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.List;

import org.junit.jupiter.api.Test;

import com.acme.gateway.config.GrayProperties;

class GrayTagFilterTest {

    @Test
    void whitelistedUsersAreTaggedAndForgedHeadersIgnored() {
        GrayTagFilter strict = new GrayTagFilter(new GrayProperties(true, "v2", List.of("1001"), 0, false));
        assertThat(strict.resolveTag(null, "1001")).isEqualTo("v2");
        assertThat(strict.resolveTag(null, "2002")).isNull();
        assertThat(strict.resolveTag("v9", "2002")).isNull();

        GrayTagFilter testEnv = new GrayTagFilter(new GrayProperties(false, null, List.of(), 0, true));
        assertThat(testEnv.resolveTag("v9", "")).isEqualTo("v9");
    }
}
