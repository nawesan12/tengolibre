package ar.com.tengolibre.app.nucleo

import android.content.Context
import android.util.Base64
import android.util.Log
import com.google.crypto.tink.Aead
import com.google.crypto.tink.KeyTemplates
import com.google.crypto.tink.RegistryConfiguration
import com.google.crypto.tink.aead.AeadConfig
import com.google.crypto.tink.integration.android.AndroidKeysetManager

/** La sesión de Tengo Libre en el teléfono (el Keychain de iOS): cifrada con Tink (AES-GCM, la clave vive en el
 *  Android Keystore), sólo en este aparato (sin copia de seguridad). Los widgets corren en el mismo proceso. */
object Llavero {
    const val servicio = "ar.com.tengolibre.app.sesion"
    internal var contexto: Context? = null
    private val prefs get() = contexto?.getSharedPreferences(servicio, Context.MODE_PRIVATE)

    // Si el Keystore falla (pasa en algunos equipos), no queda en null para siempre: se reintenta en el próximo uso.
    private var cifrador: Aead? = null
    private val aead: Aead?
        @Synchronized get() {
            cifrador?.let { return it }
            val c = contexto ?: return null
            return runCatching {
                AeadConfig.register()
                AndroidKeysetManager.Builder()
                    .withSharedPref(c, "tl_llavero", "tl_llaves")
                    .withKeyTemplate(KeyTemplates.get("AES256_GCM"))
                    .withMasterKeyUri("android-keystore://tl_maestra")
                    .build().keysetHandle.getPrimitive(RegistryConfiguration.get(), Aead::class.java)
            }.onFailure { Log.w("Llavero", "El Keystore no respondió", it) }.getOrNull().also { cifrador = it }
        }

    fun guardar(valor: String, cuenta: String) {
        val a = aead ?: run { Log.w("Llavero", "No se pudo guardar la sesión: sin Keystore"); return }
        prefs?.edit()?.putString(cuenta, Base64.encodeToString(a.encrypt(valor.toByteArray(), cuenta.toByteArray()), Base64.NO_WRAP))?.apply()
    }

    fun leer(cuenta: String): String? {
        val t = prefs?.getString(cuenta, null) ?: return null
        return runCatching { String(aead!!.decrypt(Base64.decode(t, Base64.NO_WRAP), cuenta.toByteArray())) }.getOrNull()
    }

    fun borrar(cuenta: String) { prefs?.edit()?.remove(cuenta)?.apply() }
}
