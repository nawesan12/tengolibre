<!-- Las horas libres como huecos (firma 2 del handoff, con el color del negocio): rayado que corre
	despacio; la elegida, llena. Con `href` cada hora es un link; con `nombre`, un radio de un
	formulario (`elegida` se puede enlazar). Funciona sin JavaScript en los dos casos. -->
<script lang="ts">
	import type { Horario } from './tipos.ts';

	let {
		horarios,
		href,
		nombre,
		actual = '',
		elegida = $bindable(''),
		chica = false
	}: {
		horarios: Horario[];
		href?: (h: Horario) => string;
		nombre?: string;
		/** Con links: el inicio de la hora elegida. */
		actual?: string;
		elegida?: string;
		chica?: boolean;
	} = $props();
	const igual = (a: string, b: string) => !!a && !!b && Date.parse(a) === Date.parse(b);
</script>

<ul class="horas" class:chica>
	{#each horarios as h, i (h.inicio)}
		<li style:--i={i}>
			{#if href}
				<a class="hora" href={href(h)} data-sveltekit-noscroll aria-current={igual(actual, h.inicio) ? 'true' : undefined}>{h.hora}</a>
			{:else}
				<label class="hora hora-op"><input type="radio" name={nombre} value={h.inicio} required bind:group={elegida} />{h.hora}</label>
			{/if}
		</li>
	{/each}
</ul>

<style>
	.horas {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(76px, 1fr));
		gap: 6px;
	}
	.chica {
		grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
	}
	.hora {
		position: relative;
		display: grid;
		place-items: center;
		height: 48px;
		border-radius: 12px;
		background: repeating-linear-gradient(-45deg, var(--w-raya) 0 2px, transparent 2px 9px), var(--w-tint);
		background-size: 28.28px 28.28px, auto;
		box-shadow: inset 0 0 0 1.5px var(--w-troquel);
		color: var(--w-ink);
		font: 700 15px var(--tl-font);
		font-variant-numeric: tabular-nums;
		letter-spacing: -0.02em;
		text-decoration: none;
		cursor: pointer;
		transition: transform 0.15s;
		animation:
			tl-pop 0.35s calc(var(--i) * 25ms) both,
			tl-raya-h 4s linear infinite;
	}
	.chica .hora {
		height: 48px;
		font-size: 14px;
	}
	.hora:hover {
		transform: translateY(-2px);
	}
	.hora:active {
		transform: scale(0.94);
	}
	.hora[aria-current],
	.hora:has(input:checked) {
		background: var(--w-pri);
		color: var(--w-on);
		box-shadow: none;
		animation: none;
	}
	.hora:has(input:focus-visible) {
		outline: 3px solid var(--tl-menta);
		outline-offset: 2px;
	}
	.hora input {
		position: absolute;
		opacity: 0;
		width: 1px;
		height: 1px;
		margin: 0;
	}
	@keyframes tl-pop {
		0% {
			opacity: 0;
			transform: scale(0.7);
		}
		70% {
			transform: scale(1.05);
		}
		100% {
			opacity: 1;
			transform: none;
		}
	}
	@keyframes tl-raya-h {
		to {
			background-position: 28.28px 0, 0 0;
		}
	}
</style>
