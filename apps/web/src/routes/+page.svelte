<script lang="ts">
	// Textos de la ficha de Tengo Libre en el sitio de Grupo Vimo (web/sitio/fuente/Ficha.dc.html).
	const funciones = [
		['Reservas online', 'El cliente elige servicio, profesional y horario.'],
		['Recordatorios automáticos', 'Menos ausencias, sin mandar mensajes a mano.'],
		['Señas con Mercado Pago', 'Una seña o el servicio completo.'],
		['Agenda por profesional', 'Cada uno con sus horarios y servicios.'],
		['Ficha de cada cliente', 'Qué se hizo, cuándo vino y cuándo vuelve.'],
		['Membresías', 'Abonos mensuales, como cuatro cortes al mes.'],
		['Cupones y referidos', 'Para llenar los horarios flojos.'],
		['Sucursales y marca propia', 'Varias sedes, con tu nombre y tus colores.']
	];
</script>

<svelte:head>
	<title>Tengo Libre · Turnos online</title>
	<meta
		name="description"
		content="Turnos online para barberías, peluquerías, salones de uñas y centros de estética. De Grupo Vimo."
	/>
</svelte:head>

<header class="contenedor fila cabecera">
	<strong>Tengo Libre</strong>
	<nav class="fila" aria-label="Sitio">
		<a href="/explorar">Explorar</a>
		<a class="boton" href="/ingresar">Ingresar</a>
	</nav>
</header>

<main>
	<section class="contenedor pila portada">
		<p class="etiqueta">Negocios con turnos</p>
		<h1>Que tus clientes saquen turno&nbsp;solos.</h1>
		<p class="bajada">
			El cliente reserva a cualquier hora y recibe el recordatorio antes de venir. Vos ves la agenda de cada
			profesional, cobrás señas con Mercado Pago y conocés a tus&nbsp;clientes.
		</p>
		<div class="fila">
			<a class="boton primario" href="/empezar">Probar 7 días gratis</a>
			<a class="boton" href="/explorar">Buscar un negocio</a>
			<a class="boton" href="/ingresar">Ingresar</a>
		</div>
		<p class="suave chico">Tenés 7 días gratis con todo. Después elegís un&nbsp;plan.</p>
	</section>

	<section class="contenedor pila" aria-labelledby="para-quien">
		<h2 id="para-quien">Para quién es</h2>
		<p>Barberías, peluquerías, salones de uñas y centros de&nbsp;estética.</p>
	</section>

	<section class="contenedor pila" aria-labelledby="funciones">
		<h2 id="funciones">Todo lo que pasa entre turno y&nbsp;turno.</h2>
		<ul class="grilla funciones">
			{#each funciones as [titulo, texto] (titulo)}
				<li class="tarjeta">
					<h3>{titulo}</h3>
					<p class="suave">{texto}</p>
				</li>
			{/each}
		</ul>
	</section>

	<section class="contenedor">
		<div class="tarjeta pila cierre">
			<h2>Probalo con tu negocio</h2>
			<p class="suave">Armás tu página de reservas en unos minutos y tenés 7 días con&nbsp;todo.</p>
			<div class="fila">
				<a class="boton primario" href="/empezar">Empezar gratis</a>
				<a class="boton" href="/ingresar">Ya tengo cuenta</a>
			</div>
		</div>
	</section>
</main>

<footer class="contenedor suave chico">Tengo Libre es de Grupo&nbsp;Vimo.</footer>

<style>
	.cabecera {
		justify-content: space-between;
		padding-block: 14px;
	}
	.cabecera nav a:not(.boton) {
		text-decoration: none;
		padding: 8px;
	}
	.portada {
		padding-block: 32px 40px;
	}
	.portada > :global(*) {
		max-width: 720px;
	}
	.portada .etiqueta {
		align-self: flex-start;
		margin: 0;
	}
	.portada h1 {
		font-size: clamp(30px, 6vw, 48px);
	}
	.bajada {
		font-size: 17px;
		margin: 0;
	}
	.funciones {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.funciones h3 {
		font-size: 16px;
		margin: 0 0 4px;
	}
	.funciones p {
		margin: 0;
	}
	h2 {
		margin: 0;
	}
	section p {
		margin: 0;
	}
</style>
