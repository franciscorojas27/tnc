TNC — Tiny Network Check

Una herramienta ligera de diagnóstico y sondeo de red escrita en Go. `tnc`
resuelve nombres, mide tiempos de respuesta (RTT), identifica la IP fuente y la
interfaz saliente, y exporta resultados en CSV/JSON/HTML para análisis.

¿Por qué usar `tnc`?
- Salida legible en consola con campos clave (RemoteAddress, ResolvedName,
  SourceAddress, Interface, PingRTT, Ports).
- Exportes listos para integrar en hojas de cálculo o sistemas de inventario.
- Código modular y tests básicos incluidos para facilitar mantenimiento.

Requisitos
- Go 1.20+ (proyectos desarrollados/tests con Go 1.25).

Compilar
```bash
go build -o tnc.exe
```

Uso rápido
```bash
# Escaneo básico
./tnc.exe -target example.com -ports 80,443 -format csv -out report.csv

# Escaneo de rango LAN con más concurrencia
./tnc.exe -target 192.168.1.0/24 -ports 22-23 -concurrency 50 -format json -out lan.json
```

Flags principales
- `-target`       : Host, nombre o CIDR a escanear (ej: `google.com` o `192.168.1.0/24`).
- `-ports`        : Puertos TCP (ej: `22,80,8000-8010`).
- `-udp`          : Puertos UDP (opcional).
- `-format`       : `csv` | `json` | `html` (default `csv`).
- `-out`          : Archivo de salida.
- `-concurrency`  : Número de goroutines/ workers (por defecto 20).
- `-timeout`      : Timeout en ms para operaciones de red (por defecto 400).

Export formats
- CSV: columnas fijas: `Target,IP,ResolvedName,SourceAddress,Interface,PingRTT,NetBIOS,Signing,OS,MAC,Vendor,Ports`.
  - `Ports` usa sintaxis `PROTO:PORT:STATUS:DURATION` y se separa por `;` cuando hay
    múltiples puertos.
- JSON: salida estructurada con `HostResult` y `PortResult` objetos (ideal para
  integraciones programáticas).
- HTML: reporte sencillo con valores escapados para evitar inyección.

Ejemplo de export CSV (fila):
```
example.com,93.184.216.34,example.com,10.0.0.5,Ethernet,12ms,,,,,TCP:80:Open:15ms;TCP:443:Closed:5ms
```

Notas importantes
- La inferencia de sistema operativo (`OS`) se basa en TTL y es conservadora;
  en redes públicas puede devolver `Unknown`.
- Respeta políticas y permisos antes de escanear redes externas.

Tests
```bash
go test ./...
```

Contribuciones
- Abre issues para fallos o mejoras. Envía PRs con pruebas y descripción.

Licencia
- MIT — ver `LICENSE`.

Soporte
- Abre un issue en el repositorio para preguntas o solicitud de cambios.

Si quieres, puedo abrir un PR o empujar estos cambios directamente a tu rama remota.
