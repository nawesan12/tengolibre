package ar.com.tengolibre.app.pantallas.ingreso

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ar.com.tengolibre.app.app.LocalEstado
import ar.com.tengolibre.app.nucleo.ClerkMail
import ar.com.tengolibre.app.pantallas.hoy.BarraHoja
import ar.com.tengolibre.app.tema.*
import kotlinx.coroutines.launch

// Clon de Pantallas/Ingreso/IngresoMail.swift.

/** «Entrar con mail» (como Corredor y la web): según la cuenta, la contraseña o un código de 6 números; si es la primera
 *  vez se crea la cuenta (con código, y la contraseña si la pide). «¿No podés entrar?» lleva a recuperar el acceso.
 *  Termina igual que Google y Apple: Estado.ingresarCon. Va adentro de una Hoja. */
@Composable
fun ColumnScope.IngresoMail(cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var email by remember { mutableStateOf("") }
    var clave by remember { mutableStateOf("") }
    var codigo by remember { mutableStateOf("") }
    var paso by remember { mutableStateOf<ClerkMail.Paso?>(null) }
    var flujo by remember { mutableStateOf(ClerkMail.configurado()) }
    var ocupado by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    var recuperando by remember { mutableStateOf(false) }

    fun otroMail() {
        paso = null
        clave = ""; codigo = ""; error = ""
        flujo = ClerkMail.configurado()
    }

    fun seguir() {
        val f = flujo ?: return
        if (ocupado) return
        if (paso is ClerkMail.Paso.claveNueva) {
            val (largo, comun) = ClerkMail.requisitos(clave)
            if (!largo || !comun) { error = "La contraseña todavía no cumple los requisitos."; return }
        }
        error = ""
        ocupado = true
        val actual = paso
        val mail = email.trim().lowercase()
        alcance.launch {
            try {
                val sig = when (actual) {
                    null -> f.empezar(mail)
                    is ClerkMail.Paso.clave -> f.entrarConClave(clave)
                    is ClerkMail.Paso.codigo -> f.confirmarCodigo(codigo)
                    is ClerkMail.Paso.claveNueva -> f.guardarClave(clave)
                    is ClerkMail.Paso.listo -> return@launch
                }
                if (sig is ClerkMail.Paso.listo) {
                    estado.ingresarCon(sig.token)
                    Haptico.exito()
                    cerrar()
                } else {
                    if (sig is ClerkMail.Paso.claveNueva) clave = ""
                    if (sig is ClerkMail.Paso.codigo) codigo = ""
                    paso = sig
                }
            } catch (e: kotlinx.coroutines.CancellationException) {
                throw e
            } catch (e: Exception) {
                error = ClerkMail.mensajeDe(e)
                Haptico.error()
            } finally {
                ocupado = false
            }
        }
    }

    BarraHoja("Entrar con mail", cerrar)
    Column(
        Modifier.weight(1f).background(C.fondo).verticalScroll(rememberScrollState()).padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        when (val p = paso) {
            null -> {
                Texto("Tu mail", Tipo.lato(30f, Peso.Black), C.tinta, tracking = (-0.8).sp)
                Texto("Si es tu primera vez, creamos tu cuenta con un código que te llega por mail.", Tipo.lato(16f), C.sub)
                Campo("", email, { email = it }, ayuda = "tu@mail.com", teclado = KeyboardType.Email)
                BotonMenta(if (ocupado) "Un segundo…" else "Seguir", { seguir() }, alto = 52.dp, conSombra = false,
                    habilitado = !ocupado && email.contains("@") && flujo != null)
            }
            else -> {
                Row(Modifier.heightIn(min = 44.dp).clickable { otroMail() }.semantics { contentDescription = "Usar otro mail" },
                    verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    Icono("chevron.left", 15f, C.mentaTexto)
                    Texto(email, Tipo.lato(15f, Peso.Bold), C.mentaTexto, lineas = 1)
                }
                when (p) {
                    is ClerkMail.Paso.clave -> {
                        Texto("Tu contraseña", Tipo.lato(30f, Peso.Black), C.tinta, tracking = (-0.8).sp)
                        Campo("", clave, { clave = it }, secreto = true)
                        BotonMenta(if (ocupado) "Entrando…" else "Entrar", { seguir() }, alto = 52.dp, conSombra = false, habilitado = !ocupado && clave.isNotEmpty())
                        Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) { BotonTexto("¿No podés entrar?", { recuperando = true }) }
                    }
                    is ClerkMail.Paso.codigo -> {
                        Texto("Revisá tu mail", Tipo.lato(30f, Peso.Black), C.tinta, tracking = (-0.8).sp)
                        Texto("Te mandamos un código a $email. ${if (p.alta) "Con él creamos tu cuenta." else "Escribilo para entrar."}", Tipo.lato(16f), C.sub)
                        CampoCodigo(codigo, { codigo = it }) { if (!ocupado) seguir() }
                        BotonMenta(if (ocupado) "Revisando…" else if (p.alta) "Crear mi cuenta" else "Entrar", { seguir() }, alto = 52.dp, conSombra = false,
                            habilitado = !ocupado && codigo.length >= 6)
                    }
                    is ClerkMail.Paso.claveNueva -> {
                        Texto("Elegí una contraseña", Tipo.lato(30f, Peso.Black), C.tinta, tracking = (-0.8).sp)
                        Texto("La vas a usar para entrar con tu mail.", Tipo.lato(16f), C.sub)
                        Campo("", clave, { clave = it }, secreto = true)
                        RequisitosClave(clave)
                        BotonMenta(if (ocupado) "Guardando…" else "Guardar y entrar", { seguir() }, alto = 52.dp, conSombra = false, habilitado = !ocupado)
                    }
                    is ClerkMail.Paso.listo -> {}
                }
            }
        }
        if (error.isNotEmpty()) Texto(error, Tipo.lato(14f, Peso.Bold), C.error)
        if (flujo == null) Texto("El ingreso con mail no está disponible en este momento. Probá de nuevo en un rato.", Tipo.lato(14f, Peso.Bold), C.error)
    }
    Hoja(recuperando, { recuperando = false }) { RecuperarAcceso(email) { recuperando = false } }
}

/** El código de 6 números que llega por mail: grande, espaciado, y sigue solo al completarlo. */
@Composable
fun CampoCodigo(codigo: String, alCambiar: (String) -> Unit, completo: () -> Unit) {
    val tinta = C.tinta
    Box(Modifier.fillMaxWidth().clip(RoundedCornerShape(14.dp)).background(C.campo).padding(horizontal = 14.dp, vertical = 13.dp)
        .semantics { contentDescription = "Código" }) {
        if (codigo.isEmpty()) Texto("123456", Tipo.lato(28f, Peso.Bold, numeros = true).copy(letterSpacing = 8.sp), C.sub.copy(alpha = 0.7f), lineas = 1)
        BasicTextField(
            codigo,
            { n ->
                val solo = n.filter { it.isDigit() }.take(6)
                alCambiar(solo)
                if (solo.length == 6 && solo != codigo) completo()
            },
            Modifier.fillMaxWidth(),
            textStyle = Tipo.lato(28f, Peso.Bold, numeros = true).copy(color = tinta, letterSpacing = 8.sp),
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword),
            cursorBrush = SolidColor(C.mentaTexto),
        )
    }
}

/** Los requisitos de la contraseña nueva, que se tildan solos mientras se escribe. */
@Composable
fun RequisitosClave(clave: String) {
    val (largo, comun) = ClerkMail.requisitos(clave)
    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        listOf(largo to "8 caracteres o más", comun to "No es una de las más usadas").forEach { (ok, texto) ->
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Icono(if (ok) "checkmark.circle.fill" else "circle", 16f, if (ok) C.mentaTexto else C.sub)
                Texto(texto, Tipo.lato(14f, if (ok) Peso.Bold else Peso.Regular), if (ok) C.mentaTexto else C.sub)
            }
        }
    }
}

/** Recuperar el acceso con Clerk: un código por mail, con él la contraseña nueva, se cierran las otras sesiones y se
 *  entra. Si el mail no tiene cuenta, dice lo mismo (no se revela quién está registrado). Va adentro de una Hoja. */
@Composable
fun ColumnScope.RecuperarAcceso(emailInicial: String, cerrar: () -> Unit) {
    val estado = LocalEstado.current
    val alcance = rememberCoroutineScope()
    var email by remember { mutableStateOf(emailInicial) }
    var codigo by remember { mutableStateOf("") }
    var clave by remember { mutableStateOf("") }
    var paso by remember { mutableStateOf("mail") }
    var enviado by remember { mutableStateOf(false) }
    var enviando by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf("") }
    val flujo = remember { ClerkMail.configurado() }

    fun enviar() {
        val f = flujo ?: return
        if (enviando) return
        if (paso == "nueva") {
            val (largo, comun) = ClerkMail.requisitos(clave)
            if (!largo || !comun) { error = "La contraseña todavía no cumple los requisitos."; return }
        }
        error = ""
        enviando = true
        alcance.launch {
            try {
                suspend fun entrar(token: String) {
                    estado.ingresarCon(token)
                    Haptico.exito()
                    cerrar()
                }
                when (paso) {
                    "mail" -> {
                        val hay = f.pedirRecuperar(email.trim().lowercase())
                        enviado = true
                        if (hay) paso = "codigo"
                    }
                    "codigo", "confianza" -> {
                        val sig = f.confirmarCodigo(codigo, recuperar = paso == "codigo")
                        if (sig is ClerkMail.Paso.claveNueva) paso = "nueva"
                        if (sig is ClerkMail.Paso.listo) entrar(sig.token)
                    }
                    else -> {
                        val sig = f.guardarClave(clave)
                        if (sig is ClerkMail.Paso.listo) entrar(sig.token)
                        // Teléfono nuevo para Clerk: un código más por mail para terminar.
                        if (sig is ClerkMail.Paso.codigo) { codigo = ""; enviado = false; paso = "confianza" }
                    }
                }
            } catch (e: kotlinx.coroutines.CancellationException) {
                throw e
            } catch (e: Exception) {
                error = ClerkMail.mensajeDe(e)
                Haptico.error()
            } finally {
                enviando = false
            }
        }
    }

    BarraHoja("Recuperar el acceso", cerrar)
    Column(
        Modifier.weight(1f).background(C.fondo).verticalScroll(rememberScrollState()).padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        if (paso == "nueva") {
            Texto("Elegí una contraseña nueva", Tipo.lato(30f, Peso.Black), C.tinta, tracking = (-0.8).sp)
            Campo("", clave, { clave = it }, secreto = true)
            RequisitosClave(clave)
            if (error.isNotEmpty()) Texto(error, Tipo.lato(14f, Peso.Bold), C.error)
            BotonMenta(if (enviando) "Guardando…" else "Guardar y entrar", { enviar() }, alto = 52.dp, conSombra = false, habilitado = !enviando)
            Texto("Cerramos las otras sesiones abiertas por seguridad.", Tipo.lato(13f), C.sub, Modifier.fillMaxWidth(), alinear = androidx.compose.ui.text.style.TextAlign.Center)
        } else {
            Box(Modifier.size(56.dp).clip(CircleShape).background(C.mentaSuave), contentAlignment = Alignment.Center) { Icono("envelope.badge", 24f, C.mentaTexto) }
            Texto("¿No podés entrar?", Tipo.lato(30f, Peso.Black), C.tinta, tracking = (-0.8).sp)
            Texto("Si entrás con Google o Apple, usá ese botón: no necesitás contraseña. Si usás mail, te mandamos un código para crear una nueva.", Tipo.lato(16f), C.sub)
            if (paso == "mail") {
                Campo("", email, { email = it }, ayuda = "tu@mail.com", teclado = KeyboardType.Email)
                BotonMenta(if (enviando) "Mandando…" else "Mandarme el código", { enviar() }, alto = 52.dp, conSombra = false,
                    habilitado = !enviando && flujo != null && email.contains("@"))
            } else {
                Texto("Código que te llegó a $email", Tipo.lato(13f, Peso.Bold), C.sub)
                CampoCodigo(codigo, { codigo = it }) {}
                BotonMenta(if (enviando) "Revisando…" else "Seguir", { enviar() }, alto = 52.dp, conSombra = false, habilitado = !enviando && codigo.length >= 6)
            }
            if (error.isNotEmpty()) Texto(error, Tipo.lato(14f, Peso.Bold), C.error)
            if (enviado) {
                Row(Modifier.fillMaxWidth().clip(RoundedCornerShape(14.dp)).background(C.mentaSuave).padding(12.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Icono("envelope", 16f, C.tinta)
                    Texto("Listo: si ese mail tiene cuenta, te llega un código en un minuto. Revisá también spam.", Tipo.lato(14f), C.tinta)
                }
            }
            Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) { BotonTexto("Volver a ingresar", cerrar) }
        }
    }
}
