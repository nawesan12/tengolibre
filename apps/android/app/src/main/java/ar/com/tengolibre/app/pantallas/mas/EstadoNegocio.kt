package ar.com.tengolibre.app.pantallas.mas

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue

/** Cómo está el negocio por dentro: qué está conectado, prendido o cargado (Pantallas/Mas/EstadoNegocio.swift).
 *  Se porta entero con las secciones de Más (fase 3 de ESTADO.md); por ahora sólo lo que usa Estado. */
class EstadoNegocio {
    var cargado by mutableStateOf(false)
}
