<script lang="ts">
	import Legal from '#lib/legal/Legal.svelte';

	let { form } = $props();
</script>

<Legal titulo="Botón de arrepentimiento" actualizado="19 de noviembre de 2026">
	{#if form && 'codigo' in form && form.codigo}
		<p class="listo" role="status">
			Recibimos tu pedido. Tu código es <b class="codigo">{form.codigo}</b>: guardalo para seguirlo. Te escribimos al mail que dejaste en hasta 10 días
			hábiles y, si corresponde, te devolvemos lo&nbsp;cobrado.
		</p>
	{:else}
		<p>
			Si contrataste algo de Tengo Libre en línea (una suscripción, un pack de recordatorios o una subida de plan), podés arrepentirte dentro de los 10
			días corridos desde que lo contrataste o desde que empezó a regir, sin dar explicaciones y sin costo. No hace falta entrar a tu&nbsp;cuenta.
		</p>
		<p>
			Lo que pagaste desde el iPhone o desde Android lo cobra Apple o Google: el reembolso se pide en su tienda, pero igual podés avisarnos&nbsp;acá.
		</p>
		<form method="POST" class="form">
			<label>
				Tu mail
				<input name="email" type="email" required maxlength="254" autocomplete="email" value={form && 'valores' in form ? form.valores?.email : ''} />
			</label>
			<label>
				Tu negocio en Tengo Libre (opcional)
				<input name="negocio" maxlength="120" placeholder="Nombre o dirección, ej. el-corte" value={form && 'valores' in form ? form.valores?.negocio : ''} />
			</label>
			<label>
				Qué querés revocar (opcional)
				<textarea name="detalle" rows="3" maxlength="1000">{form && 'valores' in form ? form.valores?.detalle : ''}</textarea>
			</label>
			{#if form && 'error' in form && form.error}<p class="error" role="alert">{form.error}</p>{/if}
			<button>Pedir la revocación</button>
		</form>
	{/if}
</Legal>

<style>
	.form {
		display: grid;
		gap: 16px;
		margin-top: 24px;
	}
	label {
		display: grid;
		gap: 6px;
		font-weight: 700;
		font-size: 15px;
		color: var(--tl-tinta);
	}
	input,
	textarea {
		font: 400 16px var(--tl-font);
		padding: 12px 14px;
		border-radius: 12px;
		border: 1.5px solid var(--tl-linea-2);
		background: var(--tl-papel);
		color: var(--tl-tinta);
	}
	button {
		justify-self: start;
		min-height: 48px;
		padding: 0 22px;
		border: 0;
		border-radius: 999px;
		background: var(--tl-petroleo);
		color: #fff;
		font: 700 16px var(--tl-font);
		cursor: pointer;
	}
	.error {
		color: var(--peligro, #a3241a);
	}
	.listo {
		padding: 16px;
		border-radius: 16px;
		background: #e3f6ec;
		color: #0f3d2b;
	}
	.codigo {
		font-variant-numeric: tabular-nums;
		letter-spacing: 0.08em;
	}
</style>
