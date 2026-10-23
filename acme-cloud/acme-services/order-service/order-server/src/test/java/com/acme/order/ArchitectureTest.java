package com.acme.order;

import com.acme.framework.test.AcmeArchitectureRules;
import com.tngtech.archunit.junit.AnalyzeClasses;
import com.tngtech.archunit.junit.ArchTest;
import com.tngtech.archunit.lang.ArchRule;

@AnalyzeClasses(packages = "com.acme.order")
class ArchitectureTest {

    @ArchTest
    static final ArchRule layers = AcmeArchitectureRules.layers();

    @ArchTest
    static final ArchRule domainIsFrameworkFree = AcmeArchitectureRules.domainIsFrameworkFree();

    @ArchTest
    static final ArchRule controllersDoNotTouchPersistence = AcmeArchitectureRules.controllersDoNotTouchPersistence();
}
