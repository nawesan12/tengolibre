"""Variantes del logo nuevo de Tengo Libre a partir del ícono que hizo ChatGPT (calendario crema sobre petróleo).

Separa el calendario (crema) y el día con los trazos (menta) del fondo en degradé y arma cada pieza:
ícono de app, símbolo a color y a una tinta, ícono adaptativo de Android y favicon.
"""
import base64, io, sys
from pathlib import Path
import numpy as np
from PIL import Image

ORIGEN = Path(sys.argv[1])
SALIDA = Path(sys.argv[2])
SALIDA.mkdir(parents=True, exist_ok=True)

PETROLEO = (0x18, 0x3C, 0x44)
MENTA = (0x3C, 0xCC, 0x90)
CREMA = (255, 243, 228)
BLANCO = (255, 255, 255)

icono = Image.open(ORIGEN).convert("RGB")
a = np.asarray(icono).astype(np.float32)
R, G = a[..., 0], a[..., 1]
# Menta: G muy por encima de R (fondo ~55, crema negativo, menta ~148).
am = np.clip((G - R - 62) / (140 - 62), 0, 1)
# Crema: el rojo alto (fondo ≤ 34, menta 60, crema 255); sin lo que ya es menta.
ac = np.clip((R - 45) / (250 - 45), 0, 1) * (1 - am)

# Recorte al dibujo, cuadrado y con aire.
cubre = (ac + am) > 0.05
ys, xs = np.where(cubre)
x0, x1, y0, y1 = xs.min(), xs.max(), ys.min(), ys.max()
lado = int(max(x1 - x0, y1 - y0) * 1.06)
cx, cy = (x0 + x1) // 2, (y0 + y1) // 2
caja = (cx - lado // 2, cy - lado // 2, cx - lado // 2 + lado, cy - lado // 2 + lado)


def simbolo(tinta, acento):
    """El símbolo sobre transparente: el calendario en `tinta` y el día con los trazos en `acento`."""
    rgb = np.zeros(a.shape, np.float32)
    alfa = np.clip(ac + am, 0, 1)
    for i in range(3):
        rgb[..., i] = np.where(alfa > 0, (tinta[i] * ac + acento[i] * am) / np.maximum(alfa, 1e-6), 0)
    rgba = np.dstack([rgb, alfa * 255]).round().astype(np.uint8)
    return Image.fromarray(rgba, "RGBA").crop(caja)


def guardar(im, nombre, lado_px=None):
    if lado_px:
        im = im.resize((lado_px, lado_px), Image.LANCZOS)
    im.save(SALIDA / nombre, optimize=True)
    return im


def svg_con(im, nombre, lado_px):
    """Un .svg que lleva el PNG adentro: los <img src="….svg"> de la web siguen andando igual."""
    b = io.BytesIO()
    im.resize((lado_px, lado_px), Image.LANCZOS).save(b, "PNG", optimize=True)
    datos = base64.b64encode(b.getvalue()).decode()
    (SALIDA / nombre).write_text(
        f'<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 100 100">'
        f'<image width="100" height="100" href="data:image/png;base64,{datos}" xlink:href="data:image/png;base64,{datos}"/></svg>\n')


# Ícono de app (iOS, la web, el launcher viejo de Android): el original, a sangre.
guardar(icono.copy(), "icono-1024.png", 1024)
for n in (48, 72, 96, 144, 192):
    guardar(icono.copy(), f"ic_launcher-{n}.png", n)
guardar(icono.copy(), "favicon_64.png", 64)
svg_con(icono, "favicon.svg", 64)
svg_con(icono, "app_icon_petroleo.svg", 256)

# El ícono con fondo menta (app_icon_menta): el símbolo en petróleo sobre menta.
fondo = Image.new("RGBA", icono.size, MENTA + (255,))
s = simbolo(PETROLEO, CREMA)
s_lado = int(icono.width * 0.70)
fondo.alpha_composite(s.resize((s_lado, s_lado), Image.LANCZOS), ((icono.width - s_lado) // 2,) * 2)
svg_con(fondo.convert("RGB"), "app_icon_menta.svg", 256)

# Símbolo: a color (petróleo + menta) y a una tinta (el día queda lleno, las otras celdas huecas).
principal = simbolo(PETROLEO, MENTA)
petroleo = simbolo(PETROLEO, PETROLEO)
blanco = simbolo(BLANCO, BLANCO)
for im, base in ((principal, "simbolo_principal"), (petroleo, "simbolo_una_tinta_petroleo"), (blanco, "simbolo_una_tinta_blanco")):
    guardar(im.copy(), base + ".png", 600)
    svg_con(im, base + ".svg", 512)

# Android adaptativo: 108 dp con el dibujo dentro de la zona segura (círculo de 66 dp).
crema_menta = simbolo(CREMA, MENTA)
for dens, n in (("mdpi", 108), ("hdpi", 162), ("xhdpi", 216), ("xxhdpi", 324), ("xxxhdpi", 432)):
    for im, base in ((crema_menta, "ic_launcher_foreground"), (blanco, "ic_launcher_monochrome")):
        lienzo = Image.new("RGBA", (n, n), (0, 0, 0, 0))
        d = round(n * 64 / 108)
        lienzo.alpha_composite(im.resize((d, d), Image.LANCZOS), ((n - d) // 2, (n - d) // 2))
        lienzo.save(SALIDA / f"{base}-{dens}.png", optimize=True)

# Los colores del degradé, para el fondo del ícono adaptativo.
print("degradé", "#%02X%02X%02X" % tuple(a[6, 6].astype(int)), "#%02X%02X%02X" % tuple(a[-7, -7].astype(int)))
print("recorte", caja)
