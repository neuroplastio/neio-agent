---
name: SoC Page
alias: soc
code: 0x11
---
# Usage Table

| Usage ID | Usage Name                   | Usage Types |
|----------|------------------------------|-------------|
| 00       | Undefined                    |             |
| 01       | SocControl                   | CA          |
| 02       | FirmwareTransfer             | CL          |
| 03       | FirmwareFileId               | DV          |
| 04       | FileOffsetInBytes            | DV          |
| 05       | FileTransferSizeMaxInBytes   | DV          |
| 06       | FilePayload                  | DV          |
| 07       | FilePayloadSizeInBytes       | DV          |
| 08       | FilePayloadContainsLastBytes | DF          |
| 09       | FileTransferStop             | DF          |
| 0A       | FileTransferTillEnd          | DF          |
| 0B-FFFF  | Reserved                     |             |
