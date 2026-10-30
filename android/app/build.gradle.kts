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

plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "dev.vfox.vfoxplus"
    compileSdk = 36

    defaultConfig {
        applicationId = "dev.vfox.vfoxplus"
        minSdk = 26
        targetSdk = 36
        // Same version as the desktop application.
        versionCode = 1
        versionName = rootDir.resolve("../mygo/VERSION").readText().trim()
    }

    buildTypes {
        release {
            // Android installs no APK without a signature. Until there is an
            // official key, use the debug key, which Gradle creates on the
            // first build: each build machine has its own.
            signingConfig = signingConfigs.getByName("debug")
            isMinifyEnabled = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    // kotlinOptions is deprecated in favour of compilerOptions, which needs a
    // newer Kotlin Gradle plugin than the pinned one; keep it until then.
    kotlinOptions {
        jvmTarget = "17"
    }
    buildFeatures {
        viewBinding = true
    }
}

// The Material Symbols of the app are the Android XML that google/material-design-icons publishes (Apache-2.0), imported unchanged
// as dependencies of the commit below (settings.gradle.kts has the repository) and put among the resources by the task: nothing of
// them is committed. Each one is the resource, the symbol, and the variant of the file when it is not the outlined one.
val materialSymbolsCommit = "10ecc14790612ca8fd164e3f4ffe9c16fc50d5b7"
val materialSymbols = listOf(
    Triple("ic_close", "close", null),
    Triple("ic_copy", "content_copy", null),
    Triple("ic_forward", "arrow_forward", null),
    Triple("ic_help", "help", null),
    Triple("ic_history", "history", null),
    Triple("ic_keyboard", "keyboard", null),
    Triple("ic_link", "link", null),
    Triple("ic_lock", "lock", null),
    Triple("ic_paste", "content_paste", null),
    Triple("ic_retry", "replay", null),
    Triple("ic_scan_qr", "qr_code_scanner", null),
    Triple("ic_torch_off", "flashlight_off", null),
    Triple("ic_torch_on_fill", "flashlight_on", "fill1"),
    Triple("ic_unreachable", "cloud_off", null),
)
val symbolFiles by configurations.creating {
    isCanBeConsumed = false
    isCanBeResolved = true
}

// Puts the files that Gradle imported among the resources, under the names that the code uses.
abstract class CopyMaterialSymbols : DefaultTask() {
    @get:InputFiles
    @get:PathSensitive(PathSensitivity.NAME_ONLY)
    abstract val symbols: ConfigurableFileCollection

    // The file in the cache of Gradle (SYMBOL-COMMIT[-VARIANT].xml) and the resource that it becomes.
    @get:Input
    abstract val names: MapProperty<String, String>

    // The root of a resource directory, which the Android plugin chooses: the files go to drawable/ in it.
    @get:OutputDirectory
    abstract val outputDir: DirectoryProperty

    @TaskAction
    fun copy() {
        val drawable = outputDir.get().asFile.resolve("drawable")
        drawable.deleteRecursively()
        drawable.mkdirs()
        val resource = names.get()
        symbols.files.forEach { it.copyTo(drawable.resolve(resource.getValue(it.name) + ".xml")) }
    }
}

val symbolFileNames = materialSymbols.associate { (name, symbol, variant) ->
    "$symbol-$materialSymbolsCommit${variant?.let { "-$it" } ?: ""}.xml" to name
}
// One task for each variant, registered as the source of its generated resources, so that the Android plugin runs it before it
// merges them: a directory that is only added to the sources of the app would not make Gradle run the task.
androidComponents.onVariants { variant ->
    val copy = tasks.register<CopyMaterialSymbols>("copyMaterialSymbols${variant.name.replaceFirstChar { it.uppercase() }}") {
        symbols.from(symbolFiles)
        names.set(symbolFileNames)
    }
    variant.sources.res?.addGeneratedSourceDirectory(copy, CopyMaterialSymbols::outputDir)
}

dependencies {
    materialSymbols.forEach { (_, symbol, variant) ->
        symbolFiles("symbols:$symbol:$materialSymbolsCommit${variant?.let { ":$it" } ?: ""}@xml")
    }
    implementation("androidx.core:core-ktx:1.15.0")
    implementation("androidx.appcompat:appcompat:1.7.0")
    implementation("androidx.fragment:fragment-ktx:1.8.9")
    implementation("androidx.swiperefreshlayout:swiperefreshlayout:1.1.0")
    implementation("com.google.android.material:material:1.14.0")
    // What Material 1.14.0 builds on, which the app uses itself: springs and the shape library.
    implementation("androidx.dynamicanimation:dynamicanimation:1.1.0")
    implementation("androidx.graphics:graphics-shapes:1.0.1")
    // CameraX scans the terminal QR code; zxing decodes it. Both Apache-2.0.
    implementation("androidx.camera:camera-core:1.6.2")
    implementation("androidx.camera:camera-camera2:1.6.2")
    implementation("androidx.camera:camera-lifecycle:1.6.2")
    implementation("androidx.camera:camera-view:1.6.2")
    implementation("com.google.zxing:core:3.5.3")
}
