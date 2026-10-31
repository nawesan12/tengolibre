<script lang="ts">
	import { ROLES } from '#lib/formato.ts';
	let { data } = $props();
	const duenia = $derived(data.negocios.some((n) => n.rol === 'duenio'));
	const inicial = $derived((data.persona.nombre || data.persona.email || '?').trim().charAt(0).toUpperCase());
</script>

<svelte:head><title>Mi cuenta · Tengo Libre</title></svelte:head>

<main class="contenedor pila" style="max-width: 720px">
	<h1>Mi cuenta</h1>

	<section class="tarjeta fila persona" aria-label="Tus datos">
		{#if data.persona.foto}
			<img src={data.persona.foto} alt="" width="48" height="48" />
		{:else}
			<span class="avatar" aria-hidden="true">{inicial}</span>
		{/if}
		<div class="quien">
			{#if data.persona.nombre}<strong>{data.persona.nombre}</strong>{/if}
			<span class="suave">{data.persona.email}</span>
		</div>
		<form method="POST" action="/salir"><button class="boton">Salir</button></form>
	</section>

	<section class="tarjeta pila">
		<h2>Tus negocios</h2>
		{#if data.negocios.length === 0}
			<p class="suave">Todavía no tenés un negocio. Creá el tuyo y probalo 7&nbsp;días gratis.</p>
		{:else}
			<ul class="lista">
				{#each data.negocios as n (n.id)}
					<li>
						<div class="quien">
							<strong>{n.nombre}</strong>
							<span class="suave chico">
								{ROLES[n.rol] ?? n.rol} · <a href={`/${n.slug}`} target="_blank" rel="noopener">/{n.slug}</a>
								{#if n.id === data.elegido}· <span class="etiqueta">El último que abriste</span>{/if}
							</span>
						</div>
						<form method="POST" action="/panel/negocio">
							<input type="hidden" name="id" value={n.id} />
							<button class="boton primario" aria-label={`Entrar a ${n.nombre}`}>Entrar</button>
						</form>
					</li>
				{/each}
			</ul>
		{/if}
		<div class="fila">
			<a class="boton" href="/empezar">Crear otro negocio</a>
			{#if duenia}<a class="boton" href="/panel/sucursales">Sucursales</a>{/if}
		</div>
	</section>
</main>

<style>
	.persona {
		flex-wrap: nowrap;
	}
	.persona img,
	.avatar {
		width: 48px;
		height: 48px;
		border-radius: 50%;
		flex: none;
	}
	.avatar {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		background: var(--marca-suave);
		font-weight: 650;
	}
	.quien {
		display: flex;
		flex-direction: column;
		min-width: 0;
		flex: 1;
		overflow-wrap: anywhere;
	}
	.lista {
		list-style: none;
		margin: 0;
		padding: 0;
	}
	.lista li {
		display: flex;
		gap: 12px;
		align-items: center;
		justify-content: space-between;
		padding: 10px 0;
		border-bottom: 1px solid var(--borde);
	}
</style>
