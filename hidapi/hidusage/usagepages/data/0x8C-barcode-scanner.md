---
name: Barcode Scanner Page
alias: bar
code: 0x8C
---
# Usage Table

| Usage ID | Usage Name                                   | Usage Types |
|----------|----------------------------------------------|-------------|
| 00       | Undefined                                    |             |
| 01       | Barcode  Badge  Reader                       | CA          |
| 02       | Barcode  Scanner                             | CA          |
| 03       | Dumb  Bar  Code  Scanner                     | CA          |
| 04       | Cordless  Scanner  Base                      | CA          |
| 05       | Bar  Code  Scanner  Cradle                   | CA          |
| 06-0F    | Reserved                                     |             |
| 10       | Attribute  Report                            | CL          |
| 11       | Settings  Report                             | CL          |
| 12       | Scanned  Data  Report                        | CL          |
| 13       | Raw  Scanned  Data  Report                   | CL          |
| 14       | Trigger  Report                              | CL          |
| 15       | Status  Report                               | CL          |
| 16       | UPC/EAN  Control  Report                     | CL          |
| 17       | EAN  2/3  Label  Control  Report             | CL          |
| 18       | Code  39  Control  Report                    | CL          |
| 19       | Interleaved  2  of  5  Control  Report       | CL          |
| 1A       | Standard  2  of  5  Control  Report          | CL          |
| 1B       | MSI  Plessey  Control  Report                | CL          |
| 1C       | Codabar  Control  Report                     | CL          |
| 1D       | Code  128  Control  Report                   | CL          |
| 1E       | Misc  1D  Control  Report                    | CL          |
| 1F       | 2D  Control  Report                          | CL          |
| 20-2F    | Reserved                                     |             |
| 30       | Aiming/Pointer Mode                          | SF          |
| 31       | Bar Code Present Sensor                      | SF          |
| 32       | Class 1A Laser                               | SF          |
| 33       | Class 2 Laser                                | SF          |
| 34       | Heater Present                               | SF          |
| 35       | Contact Scanner                              | SF          |
| 36       | Electronic Article Surveillance Notification | SF          |
| 37       | Constant Electronic Article Surveillance     | SF          |
| 38       | Error Indication                             | SF          |
| 39       | Fixed Beeper                                 | SF          |
| 3A       | Good Decode Indication                       | SF          |
| 3B       | Hands Free Scanning                          | SF          |
| 3C       | Intrinsically Safe                           | SF          |
| 3D       | Klasse Eins Laser                            | SF          |
| 3E       | Long Range Scanner                           | SF          |
| 3F       | Mirror Speed Control                         | SF          |
| 40       | Not On File Indication                       | SF          |
| 41       | Programmable Beeper                          | SF          |
| 42       | Triggerless                                  | SF          |
| 43       | Wand                                         | SF          |
| 44       | Water Resistant                              | SF          |
| 45       | Multi-Range Scanner                          | SF          |
| 46       | Proximity Sensor                             | SF          |
| 47-4C    | Reserved                                     |             |
| 4D       | Fragment Decoding                            | DF          |
| 4E       | Scanner Read Confidence                      | DV          |
| 4F       | Data  Prefix                                 | NAry        |
| 50       | Prefix AIMI                                  | Sel         |
| 51       | Prefix None                                  | Sel         |
| 52       | Prefix Proprietary                           | Sel         |
| 53-54    | Reserved                                     |             |
| 55       | Active Time                                  | DV          |
| 56       | Aiming Laser Pattern                         | DF          |
| 57       | Bar Code Present                             | OOC         |
| 58       | Beeper State                                 | OOC         |
| 59       | Laser On Time                                | DV          |
| 5A       | Laser State                                  | OOC         |
| 5B       | Lockout Time                                 | DV          |
| 5C       | Motor State                                  | OOC         |
| 5D       | Motor Timeout                                | DV          |
| 5E       | Power On Reset Scanner                       | DF          |
| 5F       | Prevent Read of Barcodes                     | DF          |
| 60       | Initiate Barcode Read                        | DF          |
| 61       | Trigger State                                | OOC         |
| 62       | Trigger  Mode                                | NAry        |
| 63       | Trigger Mode Blinking Laser On               | Sel         |
| 64       | Trigger Mode Continuous Laser On             | Sel         |
| 65       | Trigger Mode Laser on while Pulled           | Sel         |
| 66       | Trigger Mode Laser stays on after release    | Sel         |
| 67-6C    | Reserved                                     |             |
| 6D       | Commit Parameters to NVM                     | DF          |
| 6E       | Parameter Scanning                           | DF          |
| 6F       | Parameters Changed                           | OOC         |
| 70       | Set parameter default values                 | DF          |
| 71-74    | Reserved                                     |             |
| 75       | Scanner In Cradle                            | OOC         |
| 76       | Scanner In Range                             | OOC         |
| 77-79    | Reserved                                     |             |
| 7A       | Aim Duration                                 | DV          |
| 7B       | Good Read Lamp Duration                      | DV          |
| 7C       | Good Read Lamp Intensity                     | DV          |
| 7D       | Good Read LED                                | DF          |
| 7E       | Good Read Tone Frequency                     | DV          |
| 7F       | Good Read Tone Length                        | DV          |
| 80       | Good Read Tone Volume                        | DV          |
| 81-81    | Reserved                                     |             |
| 82       | No Read Message                              | DF          |
| 83       | Not on File Volume                           | DV          |
| 84       | Powerup Beep                                 | DF          |
| 85       | Sound Error Beep                             | DF          |
| 86       | Sound Good Read Beep                         | DF          |
| 87       | Sound Not On File Beep                       | DF          |
| 88       | Good  Read  When  to  Write                  | NAry        |
| 89       | GRWTI After Decode                           | Sel         |
| 8A       | GRWTI Beep/Lamp after transmit               | Sel         |
| 8B       | GRWTI No Beep/Lamp use at all                | Sel         |
| 8C-90    | Reserved                                     |             |
| 91       | Bookland EAN                                 | DF          |
| 92       | Convert EAN 8 to 13 Type                     | DF          |
| 93       | Convert UPC A to EAN-13                      | DF          |
| 94       | Convert UPC-E to A                           | DF          |
| 95       | EAN-13                                       | DF          |
| 96       | EAN-8                                        | DF          |
| 97       | EAN-99 128 Mandatory                         | DF          |
| 98       | EAN-99 P5/128 Optional                       | DF          |
| 99       | Enable EAN Two Label                         | DF          |
| 9A       | UPC/EAN                                      | DF          |
| 9B       | UPC/EAN Coupon Code                          | DF          |
| 9C       | UPC/EAN Periodicals                          | DV          |
| 9D       | UPC-A                                        | DF          |
| 9E       | UPC-A with 128 Mandatory                     | DF          |
| 9F       | UPC-A with 128 Optional                      | DF          |
| A0       | UPC-A with P5 Optional                       | DF          |
| A1       | UPC-E                                        | DF          |
| A2       | UPC-E1                                       | DF          |
| A3-A8    | Reserved                                     |             |
| A9       | Periodical                                   | NAry        |
| AA       | Periodical Auto-Discriminate +2              | Sel         |
| AB       | Periodical Only Decode with +2               | Sel         |
| AC       | Periodical Ignore +2                         | Sel         |
| AD       | Periodical Auto-Discriminate +5              | Sel         |
| AE       | Periodical Only Decode with +5               | Sel         |
| AF       | Periodical Ignore +5                         | Sel         |
| B0       | Check                                        | NAry        |
| B1       | Check Disable Price                          | Sel         |
| B2       | Check Enable 4 digit Price                   | Sel         |
| B3       | Check Enable 5 digit Price                   | Sel         |
| B4       | Check Enable European 4 digit Price          | Sel         |
| B5       | Check Enable European 5 digit Price          | Sel         |
| B6-B6    | Reserved                                     |             |
| B7       | EAN Two Label                                | DF          |
| B8       | EAN Three Label                              | DF          |
| B9       | EAN 8 Flag Digit 1                           | DV          |
| BA       | EAN 8 Flag Digit 2                           | DV          |
| BB       | EAN 8 Flag Digit 3                           | DV          |
| BC       | EAN 13 Flag Digit 1                          | DV          |
| BD       | EAN 13 Flag Digit 2                          | DV          |
| BE       | EAN 13 Flag Digit 3                          | DV          |
| BF       | Add EAN 2/3 Label Definition                 | DF          |
| C0       | Clear all EAN 2/3 Label Definitions          | DF          |
| C1-C2    | Reserved                                     |             |
| C3       | Codabar                                      | DF          |
| C4       | Code 128                                     | DF          |
| C5-C6    | Reserved                                     |             |
| C7       | Code 39                                      | DF          |
| C8       | Code 93                                      | DF          |
| C9       | Full ASCII Conversion                        | DF          |
| CA       | Interleaved 2 of 5                           | DF          |
| CB       | Italian Pharmacy Code                        | DF          |
| CC       | MSI/Plessey                                  | DF          |
| CD       | Standard 2 of 5 IATA                         | DF          |
| CE       | Standard 2 of 5                              | DF          |
| CF-D2    | Reserved                                     |             |
| D3       | Transmit Start/Stop                          | DF          |
| D4       | Tri-Optic                                    | DF          |
| D5       | UCC/EAN-128                                  | DF          |
| D6       | Check  Digit                                 | NAry        |
| D7       | Check Digit Disable                          | Sel         |
| D8       | Check Digit Enable Interleaved 2 of 5 OPCC   | Sel         |
| D9       | Check Digit Enable Interleaved 2 of 5 USS    | Sel         |
| DA       | Check Digit Enable Standard 2 of 5 OPCC      | Sel         |
| DB       | Check Digit Enable Standard 2 of 5 USS       | Sel         |
| DC       | Check Digit Enable One MSI Plessey           | Sel         |
| DD       | Check Digit Enable Two MSI Plessey           | Sel         |
| DE       | Check Digit Codabar Enable                   | Sel         |
| DF       | Check Digit Code 39 Enable                   | Sel         |
| E0-EF    | Reserved                                     |             |
| F0       | Transmit  Check  Digit                       | NAry        |
| F1       | Disable Check Digit Transmit                 | Sel         |
| F2       | Enable Check Digit Transmit                  | Sel         |
| F3-FA    | Reserved                                     |             |
| FB       | Symbology Identifier 1                       | DV          |
| FC       | Symbology Identifier 2                       | DV          |
| FD       | Symbology Identifier 3                       | DV          |
| FE       | Decoded Data                                 | DV          |
| FF       | Decode Data Continued                        | DF          |
| 100      | Bar Space Data                               | DV          |
| 101      | Scanner Data Accuracy                        | DV          |
| 102      | Raw  Data  Polarity                          | NAry        |
| 103      | Polarity Inverted Bar Code                   | Sel         |
| 104      | Polarity Normal Bar Code                     | Sel         |
| 105-105  | Reserved                                     |             |
| 106      | Minimum Length to Decode                     | DV          |
| 107      | Maximum Length to Decode                     | DV          |
| 108      | Discrete Length to Decode 1                  | DV          |
| 109      | Discrete Length to Decode 2                  | DV          |
| 10A      | Data  Length  Method                         | NAry        |
| 10B      | DL Method Read any                           | Sel         |
| 10C      | DL Method Check in Range                     | Sel         |
| 10D      | DL Method Check for Discrete                 | Sel         |
| 10E-10F  | Reserved                                     |             |
| 110      | Aztec Code                                   | DF          |
| 111      | BC412                                        | DF          |
| 112      | Channel Code                                 | DF          |
| 113      | Code 16                                      | DF          |
| 114      | Code 32                                      | DF          |
| 115      | Code 49                                      | DF          |
| 116      | Code One                                     | DF          |
| 117      | Colorcode                                    | DF          |
| 118      | Data Matrix                                  | DF          |
| 119      | MaxiCode                                     | DF          |
| 11A      | MicroPDF                                     | DF          |
| 11B      | PDF-417                                      | DF          |
| 11C      | PosiCode                                     | DF          |
| 11D      | QR Code                                      | DF          |
| 11E      | SuperCode                                    | DF          |
| 11F      | UltraCode                                    | DF          |
| 120      | USD-5 (Slug Code)                            | DF          |
| 121      | VeriCode                                     | DF          |
| 122-FFFF | Reserved                                     |             |
