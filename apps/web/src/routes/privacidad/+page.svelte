<script lang="ts">
	import Legal from '#lib/legal/Legal.svelte';
</script>

<Legal titulo="Política de privacidad" actualizado="12 de noviembre de 2026">
	<p>
		Tengo Libre es un sistema de turnos online para negocios de servicios. Acá contamos qué datos guardamos, para qué, dónde están y qué
		podés hacer con&nbsp;ellos.
	</p>

	<h2>Quién es el responsable</h2>
	<p>
		Tengo Libre es una marca de Grupo Vimo. El responsable es <b>Nahuel Santillán</b>, CUIT 20-44859234-1, con domicilio en Mar del Plata,
		provincia de Buenos Aires, Argentina. Para todo lo que tenga que ver con tus datos, escribí a
		<a href="mailto:privacidad@tengolibre.com.ar">privacidad@tengolibre.com.ar</a>.
	</p>
	<p>
		<b>Los datos de los clientes de cada negocio son de ese negocio.</b> El negocio es el responsable de su base de clientes y decide qué
		carga y para qué; Tengo Libre la guarda y la procesa por su cuenta, como encargado (art. 25 de la Ley 25.326), sólo para darle el
		servicio. De las cuentas del panel y de los datos del negocio, el responsable somos&nbsp;nosotros.
	</p>

	<h2>Qué datos guardamos</h2>
	<ul>
		<li><b>De quien usa el panel o la app</b>: nombre, email y foto de la cuenta con la que entra (Google, Apple o su mail, a través de Clerk).</li>
		<li><b>Del negocio</b>: nombre, dirección, contacto, servicios, horarios, equipo, fotos y lo que se carga para la página de reservas.</li>
		<li>
			<b>De quien reserva</b>: nombre, teléfono, email y los turnos que pidió, con la nota que deje; si quiere, su fecha de nacimiento (para
			saludarlo ese día). También las reseñas que escribe.
		</li>
		<li>
			<b>Lo que el negocio anota de sus clientes</b>: notas internas del equipo, etiquetas, visitas, ausencias y lo que gastó, las membresías
			(plan y visitas que le quedan) y la caja (lo que se cobró en el local, con qué medio y qué productos).
		</li>
		<li>
			<b>Penalidades por ausencias</b>: si el negocio lo activa, quien falta a varios turnos queda sin poder reservar online por un tiempo. Lo
			decide el sistema solo, según la regla que elige el negocio; el negocio la puede levantar cuando quiera y el cliente puede pedírselo.
		</li>
		<li><b>Lista de espera</b>: nombre, email, teléfono y el momento que le sirve a quien se anota, para ofrecerle un lugar que se libere.</li>
		<li><b>Del equipo</b>: los días libres que pide cada profesional (fechas y un motivo general, sin datos de salud).</li>
		<li><b>Pagos</b>: las señas y suscripciones se cobran con Mercado Pago, la App Store o Google Play. No vemos ni guardamos los datos de tarjetas.</li>
		<li><b>Avisos</b>: si los activás, el identificador para mandar notificaciones a tu teléfono o navegador.</li>
	</ul>

	<h2>Para qué los usamos</h2>
	<p>
		Sólo para que el sistema funcione: tomar y gestionar reservas, mandar confirmaciones y recordatorios, cobrar señas y suscripciones,
		mostrar reportes al negocio y mantener la seguridad. No vendemos datos ni los usamos para&nbsp;publicidad.
	</p>

	<h2>Dónde están</h2>
	<p>
		En un servidor propio contratado a Hostinger, en Brasil. El tráfico pasa por Cloudflare (Estados Unidos), que protege el sitio y lo
		hace llegar más rápido: la conexión va cifrada de tu equipo a Cloudflare y de Cloudflare a nuestro servidor, y en el medio Cloudflare ve
		lo que se manda. Todos los días se
		guarda una copia de seguridad cifrada en GitHub (Estados Unidos), que sin nuestra clave no se puede&nbsp;leer.
	</p>
	<p>
		Brasil y Estados Unidos no están entre los países que la Agencia de Acceso a la Información Pública considera con protección adecuada,
		así que esto es una transferencia internacional de datos. La hacemos sólo con los proveedores de abajo y para dar el servicio, con
		conexiones cifradas y acceso restringido a nuestro&nbsp;equipo.
	</p>

	<h2>Con quién se comparten</h2>
	<p>Con los servicios que hacen falta para funcionar, y sólo con lo que cada uno necesita:</p>
	<ul>
		<li><b>Hostinger</b> (Brasil): el servidor donde están la base y las fotos.</li>
		<li><b>Cloudflare</b> (Estados Unidos): por donde pasa el tráfico del sitio, la API y la app.</li>
		<li><b>Clerk</b> (Estados Unidos): el ingreso al panel y a la app (nombre, email y foto de la cuenta).</li>
		<li>
			<b>Google y Apple</b> (Estados Unidos): el ingreso con su cuenta, los pagos en Google Play y la App Store, y las notificaciones al
			teléfono (el texto del aviso, por ejemplo el nombre del cliente y la hora del turno).
		</li>
		<li><b>Google Fonts</b> (Estados Unidos): las tipografías de la web; Google ve la dirección IP de quien la visita.</li>
		<li><b>Mercado Pago</b> (Argentina): las señas y las suscripciones (el monto y el email de quien paga).</li>
		<li><b>WhatsApp</b> (Meta, Estados Unidos): la confirmación del turno, si el negocio la activa (teléfono del cliente y datos del turno).</li>
		<li>
			<b>Calendarios</b>: si el negocio suscribe su agenda a Google Calendar, Apple o Outlook, ese calendario recibe sus turnos con el
			nombre, el teléfono y la nota del cliente. Lo activa y lo corta el negocio.
		</li>
		<li>
			<b>Asistentes de IA</b>: si el negocio conecta su asistente (Claude, ChatGPT u otro) por MCP, ese asistente ve la agenda y los
			clientes según los permisos que el negocio le da a cada clave. Lo elige y lo revoca el negocio.
		</li>
		<li><b>GitHub</b> (Estados Unidos): las copias de seguridad diarias, cifradas.</li>
	</ul>
	<p>Cada negocio ve sólo los datos de sus&nbsp;clientes.</p>

	<h2>Cuánto tiempo</h2>
	<ul>
		<li>
			<b>Tu cuenta</b>: mientras la uses. La podés borrar cuando quieras desde la app (Más › Tu cuenta) o en
			<a href="/mi-cuenta/borrar">tengolibre.com.ar/mi-cuenta/borrar</a>: se desactiva en el momento y tus datos personales se borran a los
			30&nbsp;días.
		</li>
		<li>
			<b>Un negocio cerrado</b>: a los 90 días del cierre borramos el negocio y sus clientes. De los cobros queda sólo el registro contable,
			sin datos de&nbsp;nadie.
		</li>
		<li>
			<b>Un cliente</b>: hasta que el negocio lo borre o el cliente lo pida. Si tenía turnos pasados, quedan en la agenda sin su nombre,
			teléfono ni&nbsp;email.
		</li>
		<li>Los mails y avisos ya enviados se borran a los 60 días, y el registro de actividad del panel, a los&nbsp;180.</li>
		<li>Las copias de seguridad cifradas pueden conservar por un tiempo lo que ya se borró; sólo se usan para recuperar el sistema ante una&nbsp;falla.</li>
	</ul>

	<h2>Tus derechos</h2>
	<p>
		Podés pedir ver, corregir o borrar tus datos (Ley 25.326 de Protección de Datos Personales). Escribinos a
		<a href="mailto:privacidad@tengolibre.com.ar">privacidad@tengolibre.com.ar</a> y lo resolvemos: te respondemos en 10 días corridos para el
		acceso y en 5 días hábiles para corregir o borrar. Si sos cliente de un negocio, también se lo podés pedir al negocio, que corrige o borra
		tus datos desde su panel; si nos escribís a nosotros, lo hacemos con&nbsp;él.
	</p>
	<p>
		El titular de los datos personales tiene la facultad de ejercer el derecho de acceso a los mismos en forma gratuita a
		intervalos no inferiores a seis meses, salvo que se acredite un interés legítimo al efecto conforme lo establecido en el
		artículo 14, inciso 3 de la Ley Nº 25.326. La Agencia de Acceso a la Información Pública, en su carácter de Órgano de
		Control de la Ley Nº 25.326, tiene la atribución de atender las denuncias y reclamos que interpongan quienes resulten
		afectados en sus derechos por incumplimiento de las normas vigentes en materia de protección de datos personales.
	</p>
</Legal>
