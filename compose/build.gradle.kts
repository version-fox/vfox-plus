// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import com.github.jk1.license.filter.DependencyFilter
import com.github.jk1.license.filter.LicenseBundleNormalizer
import com.github.jk1.license.render.ReportRenderer
import com.github.jk1.license.render.TextReportRenderer
import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
    alias(libs.plugins.kotlin.jvm)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.compose)
    alias(libs.plugins.spotless)
    alias(libs.plugins.kover)
    alias(libs.plugins.license.report)
}

group = "dev.vfox.plus"
version = "0.1.0"

val shared: File = rootDir.resolve("../shared")
val generatedResources = layout.buildDirectory.dir("generated/resources/vfox-plus")

dependencies {
    implementation(compose.desktop.currentOs)
    implementation(libs.material3)
    implementation(libs.coroutines.swing)
    implementation(libs.serialization.json)
    implementation(libs.color.utilities)

    testImplementation(kotlin("test"))
    testImplementation(libs.compose.ui.test)
    testImplementation(libs.coroutines.test)
}

kotlin {
    compilerOptions {
        jvmTarget = JvmTarget.JVM_21
        // The compiler is the linter: any warning fails the build.
        allWarningsAsErrors = true
        optIn.addAll(
            "androidx.compose.material3.ExperimentalMaterial3Api",
            "androidx.compose.material3.ExperimentalMaterial3ExpressiveApi",
        )
    }
}

java {
    targetCompatibility = JavaVersion.VERSION_21
    sourceCompatibility = JavaVersion.VERSION_21
}

// ---------------------------------------------------------------------------
// Resources generated from ../shared: the translations, the icons, the icon
// fonts and the licenses of what the application bundles.
// ---------------------------------------------------------------------------

val sharedResources =
    tasks.register<Copy>("sharedResources") {
        description = "Copies the translations, the icon list and the application icon of ../shared."
        from(shared.resolve("i18n")) { into("i18n") }
        from(shared.resolve("packaging/linux/dev.vfox.VfoxPlus.svg")) { rename { "icon.svg" } }
        into(generatedResources)
    }

val symbols: Provider<String> =
    providers.environmentVariable("VFOX_PLUS_SYMBOLS").orElse(
        layout.projectDirectory
            .dir(".gradle/material-symbols")
            .asFile.absolutePath,
    )

val iconFonts =
    tasks.register<Exec>("iconFonts") {
        description = "Cuts the icons of ../shared/icons.json out of Material Symbols Rounded (needs Python 3 with fontTools)."
        val tool = shared.resolve("tools/icon_font.py")
        val fonts = generatedResources.map { it.dir("fonts") }
        inputs.file(tool)
        inputs.file(shared.resolve("icons.json"))
        inputs.property("symbols", symbols)
        outputs.dir(fonts)
        // Every icon at weight 400, grade 0 and optical size 24: one font outlined
        // and one filled, for the selected navigation destination.
        val pins = listOf("--pin", "wght=400", "--pin", "GRAD=0", "--pin", "opsz=24")
        val fontDirectory = fonts.get().asFile.absolutePath
        val symbolDirectory = symbols.get()
        commandLine(
            "bash",
            "-c",
            """
            set -e
            python3 "$tool" download "$symbolDirectory" > /dev/null
            python3 "$tool" subset --fonts "$symbolDirectory" --output "$fontDirectory/MaterialSymbolsRounded.ttf" \
              --json "$fontDirectory/icons.json" --pin FILL=0 ${pins.joinToString(" ")}
            python3 "$tool" subset --fonts "$symbolDirectory" --output "$fontDirectory/MaterialSymbolsRounded-Filled.ttf" \
              --pin FILL=1 ${pins.joinToString(" ")}
            cp "$symbolDirectory/LICENSE" "$fontDirectory/MaterialSymbolsRounded-LICENSE.txt"
            """.trimIndent(),
        )
    }

licenseReport {
    configurations = arrayOf("runtimeClasspath")
    // Typed arrays: the plugin adds a renderer of its own.
    renderers = arrayOf<ReportRenderer>(TextReportRenderer("THIRD-PARTY-NOTICES.txt"))
    filters = arrayOf<DependencyFilter>(LicenseBundleNormalizer())
    allowedLicensesFile = file("allowed-licenses.json")
}

val licenseNotices =
    tasks.register<Copy>("licenseNotices") {
        description = "Puts the licenses of the bundled libraries into the resources, for the about section."
        dependsOn(tasks.named("checkLicense"), tasks.named("generateLicenseReport"))
        from(layout.buildDirectory.file("reports/dependency-license/THIRD-PARTY-NOTICES.txt"))
        into(generatedResources.map { it.dir("licenses") })
    }

sourceSets.main {
    resources.srcDir(files(generatedResources).builtBy(sharedResources, iconFonts, licenseNotices))
}

// ---------------------------------------------------------------------------
// Tests: unit tests by default; the end-to-end tests, tagged "e2e", with
// `gradle e2eTest`.
// ---------------------------------------------------------------------------

tasks.test {
    useJUnitPlatform { excludeTags("e2e", "screenshots") }
    systemProperty("vfox.plus.shared", shared.absolutePath)
}

// The application runs in the tests of the user interface, with real
// processes: they take longer than the 60 s that runTest allows by default.
fun Test.applicationTest(tag: String) {
    group = "verification"
    testClassesDirs =
        sourceSets.test
            .get()
            .output.classesDirs
    classpath = sourceSets.test.get().runtimeClasspath
    useJUnitPlatform { includeTags(tag) }
    systemProperty("vfox.plus.shared", shared.absolutePath)
    systemProperty("kotlinx.coroutines.test.default_timeout", "10m")
    testLogging {
        events("passed", "skipped", "failed")
        showStandardStreams = true
    }
    outputs.upToDateWhen { false }
}

tasks.register<Test>("e2eTest") {
    description = "Runs the end-to-end tests against the fixtures and, with VFOX_PLUS_E2E_VFOX, the real vfox."
    applicationTest("e2e")
    // The real vfox is used only when it is named; see README.md.
    environment("VFOX_PLUS_E2E_VFOX", providers.environmentVariable("VFOX_PLUS_E2E_VFOX").getOrElse(""))
}

tasks.register<Test>("screenshots") {
    description = "Takes screenshots of every page against the fixtures, into build/screenshots."
    applicationTest("screenshots")
    systemProperty(
        "vfox.plus.screenshots",
        layout.buildDirectory
            .dir("screenshots")
            .get()
            .asFile.absolutePath,
    )
}

kover {
    // The coverage gate is about the unit tests; the tests of the user
    // interface run the application with real processes.
    currentProject {
        instrumentation { disabledForTestTasks.addAll("e2eTest", "screenshots") }
    }
    reports {
        filters {
            includes { packages("dev.vfox.plus.core", "dev.vfox.plus.state") }
        }
        verify {
            rule { minBound(85) }
        }
    }
}

// ---------------------------------------------------------------------------
// Formatting: ktlint through Spotless.
// ---------------------------------------------------------------------------

spotless {
    kotlin {
        target("src/**/*.kt")
        ktlint(libs.versions.ktlint.get())
    }
    kotlinGradle {
        target("*.gradle.kts")
        ktlint(libs.versions.ktlint.get())
    }
}

// ---------------------------------------------------------------------------
// The application.
// ---------------------------------------------------------------------------

compose.desktop {
    application {
        mainClass = "dev.vfox.plus.MainKt"
        // Lets Main.kt name the window class after the application ID; a small
        // heap and the serial collector keep the memory of the JVM low, and so
        // do the first compiler of the JIT alone (the optimizing one keeps tens
        // of megabytes of its own), and Skia on the processor: Skiko draws with
        // OpenGL by default, which maps Mesa's software OpenGL and its compiler
        // when there is no real GPU.
        jvmArgs +=
            listOf(
                "--add-opens=java.desktop/sun.awt.X11=ALL-UNNAMED",
                "-Xms32m",
                "-Xmx256m",
                "-XX:+UseSerialGC",
                "-XX:TieredStopAtLevel=1",
                "-Dskiko.renderApi=SOFTWARE",
            )
        nativeDistributions {
            packageName = "vfox-plus"
            packageVersion = version.toString()
            description = "Material 3 Expressive desktop GUI for the vfox version manager"
            vendor = "Han Li and contributors"
            licenseFile.set(rootDir.resolve("../LICENSE"))
            modules("java.instrument", "jdk.unsupported")
        }
    }
}
