/**
 * Lo que la persona va escribiendo en «Tus datos»: queda en memoria mientras navega la reserva (si
 * el horario se ocupa y elige otro, o vuelve a cambiar el día, no lo escribe de nuevo).
 */
export const borrador = $state({ nombre: '', telefono: '', email: '', nota: '' });
