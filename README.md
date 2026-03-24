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
./tnc.exe -target example.com -ports 80,443 -format csv -save report.csv

# Escaneo de rango LAN con más concurrencia
./tnc.exe -target 192.168.1.0/24 -ports 22-23 -concurrency 50 -format json -save lan.json
```


Flags principales (lista completa)
- `-ComputerName` : Host, nombre, CIDR o lista separada por comas a escanear (ej: `google.com`, `192.168.1.0/24` o `host1,host2`).
- `-port`         : Puertos TCP a probar — puede ser coma-separado o rangos (ej: `22,80,8000-8010`).
- `-udp`          : Puertos UDP a probar (coma o rangos).
- `-all`          : Usa una lista `well-known` de puertos comunes.
- `-trace`        : Ejecuta traceroute/tracert y lo incluye en el campo `Trace`.
- `-hd`           : Oculta hosts no alcanzables en la salida final.
- `-v`            : Habilita consulta de MAC/vendor (ARP) cuando sea posible.
- `-netbios`      : Ejecuta consultas NetBIOS/SMB (cuando aplica).
- `-w`            : Número máximo de workers concurrentes (por defecto `20`).
- `-timeout`      : Timeout en milisegundos para operaciones de red (por defecto `400`).
- `-Save`         : Ruta de fichero donde guardar el reporte (ej: `report.csv`).
- `-format`       : Formato de export: `csv`, `json`, `html`, `txt` (por defecto `txt`).
- `-compare`      : Archivo previo para comparar resultados y mostrar diferencias.

Notas sobre flags
- `-computerName` admite múltiples entradas separadas por comas y CIDRs.
- `-port` y `-udp` aceptan listas y rangos; los rangos se expanden internamente.
- `-w` no debe ser 0 — el valor mínimo efectivo es `1`.
- `-timeout` tiene un mínimo práctico de `100ms` para evitar timeouts demasiado bajos.

Ejemplos de uso (equivalentes y combinados)
- Escaneo simple (host único, puertos TCP 80 y 443):

```bash
./tnc.exe -computerName google.com -port 80,443 -save google.csv -format csv
```

- Escaneo UDP y TCP en un rango de puertos:

```bash
./tnc.exe -computerName example.com -port 80-82 -udp 53,123 -Save example.json -format json
```

- Escaneo de red local con más concurrencia y traceroute:

```bash
./tnc.exe -computerName 192.168.1.0/24 -port 22,80 -w 50 -trace -save lan.html -format html
```

--

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
