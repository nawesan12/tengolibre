import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.roborazzi)
}

// Igual que Config/TengoLibre.xcconfig en iOS: la app no tiene secretos (la clave publicable de Clerk es pública).
// Para apuntar a otra API sin tocar esto, poné tl.api en local.properties (no se versiona).
val locales = Properties().apply {
    val f = rootProject.file("local.properties")
    if (f.exists()) f.inputStream().use { load(it) }
}
fun local(clave: String, porDefecto: String = ""): String = locales.getProperty(clave, porDefecto)
fun cadena(v: String) = "\"" + v.replace("\\", "\\\\").replace("\"", "\\\"") + "\""

android {
    namespace = "ar.com.tengolibre.app"
    compileSdk = 36

    defaultConfig {
        applicationId = "ar.com.tengolibre.app"
        minSdk = 26
        targetSdk = 36
        versionCode = 1
        versionName = "1.0"

        buildConfigField("String", "API", cadena(local("tl.api", "https://api.tengolibre.com.ar")))
        buildConfigField("String", "WEB", cadena(local("tl.web", "https://tengolibre.com.ar")))
        buildConfigField("String", "CLERK", cadena(local("tl.clerk", "pk_live_Y2xlcmsuZ3J1cG92aW1vLmNvbS5hciQ")))
        // Firebase sin google-services.json (como Caigo): si faltan estos datos, el push queda apagado.
        buildConfigField("String", "FIREBASE_APP_ID", cadena(local("tl.firebase.appId")))
        buildConfigField("String", "FIREBASE_API_KEY", cadena(local("tl.firebase.apiKey")))
        buildConfigField("String", "FIREBASE_PROYECTO", cadena(local("tl.firebase.proyecto")))
        buildConfigField("String", "FIREBASE_REMITENTE", cadena(local("tl.firebase.remitente")))
    }

    // Firma de release: la llave de subida vive fuera del repo (~/Trabajo/vimo/credenciales-android).
    signingConfigs {
        if (local("tl.firma.archivo").isNotEmpty()) {
            create("subida") {
                storeFile = file(local("tl.firma.archivo"))
                storePassword = local("tl.firma.clave")
                keyAlias = local("tl.firma.alias", "tengolibre")
                keyPassword = local("tl.firma.clave")
            }
        }
    }

    buildTypes {
        release {
            signingConfigs.findByName("subida")?.let { signingConfig = it }
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    buildFeatures {
        compose = true
        buildConfig = true
    }
    packaging {
        resources.excludes += "/META-INF/{AL2.0,LGPL2.1}"
    }
    testOptions {
        unitTests.isReturnDefaultValues = true
        unitTests.isIncludeAndroidResources = true
    }
}

kotlin {
    compilerOptions {
        jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17)
    }
}

// Las capturas de PantallasTest (claro y oscuro) van a build/capturas: el CI las sube para compararlas con iOS.
roborazzi {
    outputDir.set(layout.buildDirectory.dir("capturas"))
}

dependencies {
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.lifecycle.runtime.compose)
    implementation(libs.androidx.lifecycle.process)

    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.ui.graphics)
    implementation(libs.compose.foundation)
    implementation(libs.compose.ui.tooling.preview)
    debugImplementation(libs.compose.ui.tooling)
    debugImplementation(libs.compose.ui.test.manifest)

    implementation(libs.kotlinx.coroutines.android)
    implementation(libs.kotlinx.coroutines.play.services)
    implementation(libs.kotlinx.serialization.json)
    implementation(libs.okhttp)
    implementation(libs.coil.compose)
    implementation(libs.coil.network.okhttp)
    implementation(libs.androidx.datastore.preferences)
    implementation(libs.tink.android)
    implementation(libs.billing)
    implementation(platform(libs.firebase.bom))
    implementation(libs.firebase.messaging)
    implementation(libs.androidx.browser)
    implementation(libs.glance.appwidget)
    implementation(libs.zxing.core)

    testImplementation(libs.junit)
    testImplementation(libs.kotlinx.coroutines.test)
    testImplementation(libs.okhttp.mockwebserver)
    testImplementation(libs.robolectric)
    testImplementation(platform(libs.compose.bom))
    testImplementation(libs.compose.ui.test.junit4)
    testImplementation(libs.roborazzi)
    testImplementation(libs.roborazzi.compose)
    testImplementation(libs.roborazzi.junit.rule)
}
