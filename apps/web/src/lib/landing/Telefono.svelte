<!-- Mockup de la página pública de un negocio en el celular (ilustrativo): servicios y horarios libres. -->
<div class="telefono" aria-hidden="true">
	<div class="pantalla">
		<div class="estado"><b>9:41</b><span class="isla"></span><i>●●● ▮</i></div>
		<div class="portada"></div>
		<div class="negocio">
			<span class="logo">L</span>
			<b>Estudio Lumi</b>
			<small>★ 4,9 · Palermo</small>
		</div>
		<div class="servicios">
			<span class="servicio elegido"><b>Corte y peinado</b><small>45 min · $ 14.000</small></span>
			<span class="servicio"><b>Color</b><small>2 h · $ 38.000</small></span>
		</div>
		<div class="libres">
			<small>Libres hoy</small>
			<div><span>10:30</span><span class="sel">11:00</span><span>12:30</span><span>16:00</span></div>
		</div>
		<i class="reservar">Reservar 11:00</i>
	</div>
</div>

<style>
	.telefono {
		padding: 7px;
		border-radius: 36px;
		background: linear-gradient(145deg, #23363a, #0b1416);
		box-shadow:
			inset 0 0 0 1px oklch(1 0 0 / 0.08),
			0 50px 80px -30px oklch(0 0 0 / 0.6);
	}
	.pantalla {
		position: relative;
		aspect-ratio: 9 / 19;
		border-radius: 30px;
		overflow: hidden;
		background: #fff;
		display: flex;
		flex-direction: column;
		font-size: 12px;
		color: #0f2328;
	}
	.estado {
		position: absolute;
		inset: 0 0 auto;
		z-index: 1;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 10px 18px 4px;
		font-size: 11px;
		color: #fff;
	}
	.estado i {
		font-style: normal;
		font-size: 8px;
		letter-spacing: 1px;
	}
	.isla {
		width: 64px;
		height: 18px;
		border-radius: 999px;
		background: #0b1416;
	}
	.portada {
		flex: 0 0 25%;
		background: url('/fotos/peluquera-con-clienta.jpg') 60% 40% / cover;
	}
	.negocio {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 1px;
		margin-top: -22px;
		padding: 0 14px 8px;
		text-align: center;
	}
	.logo {
		display: grid;
		place-items: center;
		width: 40px;
		height: 40px;
		margin-bottom: 4px;
		border-radius: 12px;
		border: 3px solid #fff;
		background: #183c44;
		color: #3ccc90;
		font-size: 15px;
		font-weight: 800;
	}
	.negocio b {
		font-size: 14px;
		letter-spacing: -0.01em;
	}
	.negocio small,
	.servicio small,
	.libres small {
		font-size: 10.5px;
		color: #4a5d61;
	}
	.servicios {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 4px 12px;
	}
	.servicio {
		display: flex;
		flex-direction: column;
		padding: 8px 10px;
		border-radius: 12px;
		background: #f2f6f6;
	}
	.servicio b {
		font-size: 12px;
	}
	.servicio.elegido {
		background: #dff6ec;
		box-shadow: inset 0 0 0 1.5px #3ccc90;
	}
	.libres {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 10px 12px 0;
	}
	.libres div {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 5px;
	}
	.libres span {
		padding: 6px 0;
		border-radius: 8px;
		background: #f2f6f6;
		font-size: 10.5px;
		font-weight: 600;
		text-align: center;
	}
	.libres .sel {
		background: #183c44;
		color: #fff;
	}
	.reservar {
		margin: auto 12px 12px;
		padding: 10px;
		border-radius: 999px;
		background: #3ccc90;
		color: #102b31;
		font-style: normal;
		font-weight: 700;
		font-size: 12px;
		text-align: center;
	}
	@media (max-width: 640px) {
		.telefono {
			padding: 5px;
			border-radius: 24px;
		}
		.pantalla {
			border-radius: 20px;
			font-size: 9px;
		}
		.estado,
		.servicio:not(.elegido),
		.negocio small,
		.libres span:nth-child(n + 4) {
			display: none;
		}
		.libres div {
			grid-template-columns: repeat(3, 1fr);
			gap: 3px;
		}
		.logo {
			width: 26px;
			height: 26px;
			border-width: 2px;
			font-size: 10px;
		}
		.negocio {
			margin-top: -14px;
			padding: 0 6px 4px;
		}
		.negocio b {
			font-size: 10px;
		}
		.servicios,
		.libres {
			padding-inline: 6px;
		}
		.servicio {
			padding: 5px 6px;
		}
		.servicio b {
			font-size: 9px;
		}
		.servicio small,
		.libres small,
		.libres span {
			font-size: 8px;
		}
		.libres span {
			padding: 4px 0;
		}
		.reservar {
			margin: auto 6px 8px;
			padding: 6px;
			font-size: 8.5px;
		}
	}
</style>
