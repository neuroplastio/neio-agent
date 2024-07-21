---
name: Magnetic Stripe Reader
code: 0x8E
---
# Usage Table

| Usage ID | Usage Name             | Usage Types |
|----------|------------------------|-------------|
| 00       | Undefined              |             |
| 01       | MSR  Device  Read-Only | CA          |
| 02-10    | Reserved               |             |
| 11       | Track 1 Length         | DV          |
| 12       | Track 2 Length         | DV          |
| 13       | Track 3 Length         | DV          |
| 14       | Track JIS Length       | DV          |
| 15-1F    | Reserved               |             |
| 20       | Track Data             | SF/DF/DV    |
| 21       | Track 1 Data           | SF/DF/DV    |
| 22       | Track 2 Data           | SF/DF/DV    |
| 23       | Track 3 Data           | SF/DF/DV    |
| 24       | Track JIS Data         | SF/DF/DV    |
| 25-FFFF  | Reserved               |             |
