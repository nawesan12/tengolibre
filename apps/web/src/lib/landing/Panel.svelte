<!--
	Mockup del panel de Tengo Libre para la landing: ilustrativo (datos inventados), hasta que haya
	capturas reales del panel diseñado. Copiado y adaptado del de Corredor. En loop: un turno pasa de
	«Pendiente» a «Confirmado» y entra uno nuevo a la agenda.
-->
<script lang="ts">
	const menu = [
		['home', 'Inicio'],
		['event_available', 'Turnos'],
		['calendar_month', 'Calendario'],
		['group', 'Clientes'],
		['payments', 'Cobros'],
		['card_membership', 'Membresías'],
		['star', 'Reseñas']
	] as const;
	// Barras de la mini serie de «turnos de la semana».
	const serie = [40, 56, 48, 72, 64, 88, 80];
	const horas = ['9:00', '10:00', '11:00', '12:00'];
	// Cada turno: desde (en medias horas desde las 9) y cuántas medias horas dura.
	const profesionales = [
		{
			nombre: 'Caro',
			c: '#3ccc90',
			turnos: [
				{ d: 0, n: 2, s: 'Color', p: 'Lucía' },
				{ d: 3, n: 1, s: 'Corte', p: 'Sofía' },
				{ d: 6, n: 2, s: 'Alisado', p: 'Ana' }
			]
		},
		{
			nombre: 'Lucas',
			c: '#8fd9b8',
			turnos: [
				{ d: 0, n: 1, s: 'Corte', p: 'Juan' },
				{ d: 1, n: 1, s: 'Barba', p: 'Tomás' },
				{ d: 5, n: 2, s: 'Corte y barba', p: 'Martín' }
			]
		},
		{
			nombre: 'Mica',
			c: '#b9e6d2',
			turnos: [
				{ d: 1, n: 2, s: 'Semipermanente', p: 'Flor' },
				{ d: 4, n: 2, s: 'Kapping', p: 'Belu' }
			]
		}
	];
</script>

<div class="ventana" aria-hidden="true">
	<div class="cromo">
		<span></span><span></span><span></span>
		<i><span class="material-symbols-rounded">lock</span>tengolibre.com.ar/panel</i>
	</div>
	<div class="app">
		<aside>
			<span class="logo"><img src="/marca-icono.png" alt="" /><b>Tengo Libre</b></span>
			{#each menu as [ic, nombre], i (ic)}
				<span class="item" class:activo={i === 1}
					><span class="material-symbols-rounded">{ic}</span><span class="nombre">{nombre}</span></span
				>
			{/each}
		</aside>
		<div class="contenido">
			<div class="arriba">
				<div class="buscar"><span class="material-symbols-rounded">search</span>Buscar clientes, turnos…</div>
				<div class="equipo">
					{#each profesionales as p (p.nombre)}<span style="--c: {p.c}">{p.nombre[0]}</span>{/each}
				</div>
			</div>
			<div class="kpis">
				<div>
					<small>Turnos de hoy</small>
					<b>18</b>
					<span class="barras">
						{#each serie as h, i (i)}<i style="height: {h}%"></i>{/each}
					</span>
				</div>
				<div>
					<small>Señas cobradas</small>
					<b>$ 86.400</b>
					<em class="sube">+12% esta semana</em>
				</div>
				<div class="ocupacion">
					<small>Agenda ocupada</small>
					<b>82%</b>
					<span class="progreso"><i style="width: 82%"></i></span>
				</div>
			</div>
			<div class="agenda">
				<div class="horas">
					{#each horas as h (h)}<small>{h}</small>{/each}
				</div>
				{#each profesionales as pro, j (pro.nombre)}
					<div class="columna">
						<small class="quien"><span style="--c: {pro.c}">{pro.nombre[0]}</span>{pro.nombre}</small>
						<div class="franja">
							{#each pro.turnos as t (t.d)}
								<div class="turno" style="--d: {t.d}; --n: {t.n}">
									<b>{t.s}</b><span>{t.p}</span>
								</div>
							{/each}
							{#if j === 1}
								<div class="turno cambia" style="--d: 3; --n: 1">
									<b>Corte</b><span>Nico</span>
									<em><span class="pendiente">Pendiente</span><span class="confirmado">Confirmado</span></em>
								</div>
							{/if}
							{#if j === 2}
								<div class="turno nuevo" style="--d: 7; --n: 1">
									<b>Manos</b><span>Vale · nuevo</span>
								</div>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		</div>
	</div>
</div>

<style>
	.material-symbols-rounded {
		font-family: 'Material Symbols Rounded';
		font-weight: normal;
		font-style: normal;
		line-height: 1;
		font-variation-settings: 'FILL' 1;
	}
	.ventana {
		--fila: 30px;
		border-radius: 18px;
		background: #fff;
		overflow: hidden;
		box-shadow:
			0 0 0 1px oklch(0.17 0.012 220 / 0.06),
			0 2px 4px oklch(0 0 0 / 0.04),
			0 60px 120px -50px oklch(0.25 0.05 210 / 0.5);
		font-size: 13px;
		color: #0f2328;
	}
	.cromo {
		height: 40px;
		display: flex;
		align-items: center;
		gap: 7px;
		padding: 0 16px;
		background: #f1f5f5;
	}
	.cromo > span {
		width: 11px;
		height: 11px;
		border-radius: 50%;
		background: #ff6159;
	}
	.cromo > span:nth-child(2) {
		background: #ffbd2e;
	}
	.cromo > span:nth-child(3) {
		background: #28c941;
	}
	.cromo i {
		display: flex;
		align-items: center;
		gap: 6px;
		margin: 0 auto;
		padding: 4px 48px;
		border-radius: 8px;
		background: #fff;
		font-style: normal;
		font-size: 12.5px;
		color: #4a5d61;
	}
	.cromo i .material-symbols-rounded {
		font-size: 13px;
		color: #183c44;
	}
	.app {
		display: grid;
		grid-template-columns: 172px 1fr;
	}
	aside {
		background: linear-gradient(180deg, #183c44, #102b31);
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 16px 12px;
		color: oklch(1 0 0 / 0.6);
	}
	.logo {
		display: flex;
		align-items: center;
		gap: 9px;
		margin: 0 0 14px 4px;
		color: #fff;
		font-size: 14px;
		letter-spacing: -0.01em;
	}
	.logo img {
		width: 28px;
		height: 28px;
		border-radius: 8px;
	}
	.item {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 7px 10px;
		border-radius: 10px;
		font-size: 13px;
		font-weight: 550;
	}
	.item .material-symbols-rounded {
		font-size: 19px;
	}
	.item.activo {
		background: oklch(1 0 0 / 0.1);
		color: #fff;
	}
	.item.activo .material-symbols-rounded {
		color: #3ccc90;
	}
	.contenido {
		min-width: 0;
		padding: 18px clamp(14px, 2.2vw, 26px) 22px;
		display: flex;
		flex-direction: column;
		gap: 16px;
		background: #f7faf9;
	}
	.arriba {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
	}
	.buscar {
		display: flex;
		align-items: center;
		gap: 8px;
		flex: 1;
		max-width: 320px;
		padding: 9px 12px;
		border-radius: 10px;
		background: #fff;
		color: #7d8f92;
		box-shadow: 0 0 0 1px oklch(0 0 0 / 0.05);
		white-space: nowrap;
		overflow: hidden;
	}
	.buscar .material-symbols-rounded {
		font-size: 17px;
	}
	.equipo {
		display: flex;
	}
	.equipo span,
	.quien span {
		display: grid;
		place-items: center;
		width: 28px;
		height: 28px;
		margin-left: -8px;
		border-radius: 50%;
		border: 2px solid #f7faf9;
		background: var(--c);
		color: #102b31;
		font-size: 11px;
		font-weight: 700;
	}
	.kpis {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 12px;
	}
	.kpis > div {
		position: relative;
		padding: 14px 16px;
		border-radius: 14px;
		background: #fff;
		box-shadow: 0 1px 2px oklch(0 0 0 / 0.05);
		display: flex;
		flex-direction: column;
		gap: 2px;
		overflow: hidden;
	}
	.kpis small {
		font-size: 12px;
		color: #4a5d61;
		white-space: nowrap;
	}
	.kpis b {
		font-size: clamp(20px, 2.4vw, 28px);
		letter-spacing: -0.03em;
		white-space: nowrap;
	}
	.sube {
		font-style: normal;
		font-size: 11.5px;
		font-weight: 600;
		color: #17805a;
	}
	.barras {
		position: absolute;
		right: 14px;
		bottom: 14px;
		height: 36px;
		display: flex;
		align-items: flex-end;
		gap: 3px;
	}
	.barras i {
		width: 6px;
		border-radius: 3px;
		background: #cdeee0;
	}
	.barras i:last-child,
	.barras i:nth-last-child(2) {
		background: #183c44;
	}
	.progreso {
		height: 6px;
		margin-top: 6px;
		border-radius: 3px;
		background: #ddf3e9;
		overflow: hidden;
	}
	.progreso i {
		display: block;
		height: 100%;
		border-radius: 3px;
		background: #3ccc90;
	}

	/* La agenda del día: una columna por profesional, filas de media hora. */
	.agenda {
		display: grid;
		grid-template-columns: 40px repeat(3, 1fr);
		gap: 10px;
		padding: 14px 14px 14px 8px;
		border-radius: 14px;
		background: #fff;
		box-shadow: 0 1px 2px oklch(0 0 0 / 0.05);
	}
	.horas {
		display: flex;
		flex-direction: column;
		padding-top: 38px;
	}
	.horas small {
		height: calc(2 * var(--fila));
		font-size: 11px;
		color: #7d8f92;
		text-align: right;
		transform: translateY(-7px);
	}
	.columna {
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.quien {
		display: flex;
		align-items: center;
		gap: 8px;
		height: 28px;
		font-size: 12.5px;
		font-weight: 650;
	}
	.quien span {
		margin-left: 0;
		width: 24px;
		height: 24px;
		border: 0;
		font-size: 10px;
	}
	.franja {
		position: relative;
		height: calc(8 * var(--fila));
		background: repeating-linear-gradient(
			180deg,
			#eef3f2 0 1px,
			transparent 1px calc(2 * var(--fila))
		);
	}
	.turno {
		position: absolute;
		left: 0;
		right: 0;
		top: calc(var(--d) * var(--fila) + 2px);
		height: calc(var(--n) * var(--fila) - 4px);
		display: flex;
		flex-direction: column;
		justify-content: center;
		padding: 0 10px;
		border-radius: 8px;
		background: #e6f1f2;
		box-shadow: inset 3px 0 0 #183c44;
		overflow: hidden;
		line-height: 1.2;
	}
	.turno b,
	.turno span {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		font-size: 11.5px;
	}
	.turno span {
		font-size: 10.5px;
		color: #4a5d61;
	}
	.columna:nth-child(3) .turno {
		background: #dff6ec;
		box-shadow: inset 3px 0 0 #3ccc90;
	}
	.turno.cambia {
		flex-direction: row;
		align-items: center;
		gap: 6px;
		justify-content: flex-start;
		animation: confirmar 6s ease-in-out infinite;
	}
	.cambia em {
		position: relative;
		margin-left: auto;
		font-style: normal;
		font-size: 9.5px;
		font-weight: 700;
	}
	.cambia em span {
		padding: 2px 6px;
		border-radius: 999px;
		font-size: 9.5px;
	}
	.pendiente {
		background: #fff;
		color: #4a5d61;
		animation: mostrar-primero 6s ease-in-out infinite;
	}
	.confirmado {
		position: absolute;
		right: 0;
		top: 50%;
		transform: translateY(-50%);
		background: #3ccc90;
		color: #102b31 !important;
		animation: mostrar-despues 6s ease-in-out infinite;
	}
	@keyframes confirmar {
		0%,
		45% {
			background: #f3f6f6;
			box-shadow: inset 3px 0 0 #b8c6c8;
		}
		55%,
		100% {
			background: #dff6ec;
			box-shadow:
				inset 3px 0 0 #3ccc90,
				0 8px 18px -10px oklch(0.6 0.13 160 / 0.6);
		}
	}
	@keyframes mostrar-primero {
		0%,
		45% {
			opacity: 1;
		}
		55%,
		100% {
			opacity: 0;
		}
	}
	@keyframes mostrar-despues {
		0%,
		45% {
			opacity: 0;
		}
		55%,
		100% {
			opacity: 1;
		}
	}
	/* El turno que entra a la agenda. */
	.turno.nuevo {
		background: #3ccc90 !important;
		box-shadow: 0 10px 22px -10px oklch(0.55 0.13 160 / 0.7) !important;
		color: #102b31;
		animation: entrar 6s cubic-bezier(0.2, 0.7, 0.2, 1) infinite;
	}
	.turno.nuevo span {
		color: #102b31;
	}
	@keyframes entrar {
		0%,
		20% {
			opacity: 0;
			transform: translateY(10px) scale(0.92);
		}
		30%,
		90% {
			opacity: 1;
			transform: none;
		}
		100% {
			opacity: 0;
		}
	}
	@media (max-width: 980px) {
		.app {
			grid-template-columns: 56px 1fr;
		}
		aside {
			align-items: center;
			padding: 16px 0;
		}
		.logo b,
		.nombre {
			display: none;
		}
		.logo {
			margin: 0 0 10px;
		}
		.item {
			padding: 8px;
		}
	}
	@media (max-width: 640px) {
		.ventana {
			--fila: 22px;
			font-size: 11px;
		}
		.cromo i {
			padding: 4px 16px;
		}
		.app {
			grid-template-columns: 40px 1fr;
		}
		aside {
			gap: 2px;
		}
		.item {
			padding: 5px;
		}
		.item .material-symbols-rounded {
			font-size: 16px;
		}
		.logo img {
			width: 24px;
			height: 24px;
		}
		.equipo,
		.barras {
			display: none;
		}
		.kpis,
		.agenda {
			gap: 6px;
		}
		.kpis {
			grid-template-columns: 1fr 1fr;
		}
		.kpis > div:nth-child(3) {
			display: none;
		}
		.kpis > div {
			padding: 8px 10px;
		}
		.kpis small {
			font-size: 10.5px;
		}
		.agenda {
			grid-template-columns: 28px repeat(2, 1fr);
			padding: 10px 10px 10px 4px;
			margin-right: 98px;
		}
		.columna:nth-child(4) {
			display: none;
		}
		.horas {
			padding-top: 34px;
		}
		.horas small {
			font-size: 9.5px;
		}
		.turno {
			padding: 0 6px;
		}
		.turno b {
			font-size: 10px;
		}
		.turno span {
			display: none;
		}
		.cambia em,
		.quien span {
			display: none;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.turno.cambia,
		.pendiente,
		.confirmado,
		.turno.nuevo {
			animation: none;
		}
		.pendiente {
			opacity: 0;
		}
	}
</style>
