---
name: Physical Input Device Page
alias: pid
code: 0x0F
---
# Usage Table

| Usage ID | Usage Name                                    | Usage Types |
|----------|-----------------------------------------------|-------------|
| 00       | Undefined                                     |             |
| 01       | Physical  Input  Device                       | CA          |
| 02-1F    | Reserved                                      |             |
| 20       | Normal                                        | DV          |
| 21       | Set  Effect  Report                           | CL          |
| 22       | Effect Parameter Block Index                  | DV          |
| 23       | Parameter Block Offset                        | DV          |
| 24       | ROM Flag                                      | DF          |
| 25       | Effect  Type                                  | NAry        |
| 26       | ET Constant-Force                             | Sel         |
| 27       | ET Ramp                                       | Sel         |
| 28       | ET Custom-Force                               | Sel         |
| 29-2F    | Reserved                                      |             |
| 30       | ET Square                                     | Sel         |
| 31       | ET Sine                                       | Sel         |
| 32       | ET Triangle                                   | Sel         |
| 33       | ET Sawtooth Up                                | Sel         |
| 34       | ET Sawtooth Down                              | Sel         |
| 35-3F    | Reserved                                      |             |
| 40       | ET Spring                                     | Sel         |
| 41       | ET Damper                                     | Sel         |
| 42       | ET Inertia                                    | Sel         |
| 43       | ET Friction                                   | Sel         |
| 44-4F    | Reserved                                      |             |
| 50       | Duration                                      | DV          |
| 51       | Sample Period                                 | DV          |
| 52       | Gain                                          | DV          |
| 53       | Trigger Button                                | DV          |
| 54       | Trigger Repeat Interval                       | DV          |
| 55       | Axes Enable                                   | US          |
| 56       | Direction Enable                              | DF          |
| 57       | Direction                                     | CL          |
| 58       | Type  Specific  Block  Offset                 | CL          |
| 59       | Block  Type                                   | NAry        |
| 5A       | Set  Envelope  Report                         | CL/SV       |
| 5B       | Attack Level                                  | DV          |
| 5C       | Attack Time                                   | DV          |
| 5D       | Fade Level                                    | DV          |
| 5E       | Fade Time                                     | DV          |
| 5F       | Set  Condition  Report                        | CL/SV       |
| 60       | Center-Point Offset                           | DV          |
| 61       | Positive Coefficient                          | DV          |
| 62       | Negative Coefficient                          | DV          |
| 63       | Positive Saturation                           | DV          |
| 64       | Negative Saturation                           | DV          |
| 65       | Dead Band                                     | DV          |
| 66       | Download  Force  Sample                       | CL          |
| 67       | Isoch Custom-Force Enable                     | DF          |
| 68       | Custom-Force  Data  Report                    | CL          |
| 69       | Custom-Force Data                             | DV          |
| 6A       | Custom-Force Vendor Defined Data              | DV          |
| 6B       | Set  Custom-Force  Report                     | CL/SV       |
| 6C       | Custom-Force Data Offset                      | DV          |
| 6D       | Sample Count                                  | DV          |
| 6E       | Set  Periodic  Report                         | CL/SV       |
| 6F       | Offset                                        | DV          |
| 70       | Magnitude                                     | DV          |
| 71       | Phase                                         | DV          |
| 72       | Period                                        | DV          |
| 73       | Set  Constant-Force  Report                   | CL/SV       |
| 74       | Set  Ramp-Force  Report                       | CL/SV       |
| 75       | Ramp Start                                    | DV          |
| 76       | Ramp End                                      | DV          |
| 77       | Effect  Operation  Report                     | CL          |
| 78       | Effect  Operation                             | NAry        |
| 79       | Op Effect Start                               | Sel         |
| 7A       | Op Effect Start Solo                          | Sel         |
| 7B       | Op Effect Stop                                | Sel         |
| 7C       | Loop Count                                    | DV          |
| 7D       | Device  Gain  Report                          | CL          |
| 7E       | Device Gain                                   | DV          |
| 7F       | Parameter  Block  Pools  Report               | CL          |
| 80       | RAM Pool Size                                 | DV          |
| 81       | ROM Pool Size                                 | SV          |
| 82       | ROM Effect Block Count                        | SV          |
| 83       | Simultaneous Effects Max                      | SV          |
| 84       | Pool Alignment                                | SV          |
| 85       | Parameter  Block  Move  Report                | CL          |
| 86       | Move Source                                   | DV          |
| 87       | Move Destination                              | DV          |
| 88       | Move Length                                   | DV          |
| 89       | Effect  Parameter  Block  Load  Report        | CL          |
| 8A-8A    | Reserved                                      |             |
| 8B       | Effect  Parameter  Block  Load  Status        | NAry        |
| 8C       | Block Load Success                            | Sel         |
| 8D       | Block Load Full                               | Sel         |
| 8E       | Block Load Error                              | Sel         |
| 8F       | Block Handle                                  | DV          |
| 90       | Effect  Parameter  Block  Free  Report        | CL          |
| 91       | Type  Specific  Block  Handle                 | CL          |
| 92       | PID  State  Report                            | CL          |
| 93-93    | Reserved                                      |             |
| 94       | Effect Playing                                | DF          |
| 95       | PID  Device  Control  Report                  | CL          |
| 96       | PID  Device  Control                          | NAry        |
| 97       | DC Enable Actuators                           | Sel         |
| 98       | DC Disable Actuators                          | Sel         |
| 99       | DC Stop All Effects                           | Sel         |
| 9A       | DC Reset                                      | Sel         |
| 9B       | DC Pause                                      | Sel         |
| 9C       | DC Continue                                   | Sel         |
| 9D-9E    | Reserved                                      |             |
| 9F       | Device Paused                                 | DF          |
| A0       | Actuators Enabled                             | DF          |
| A1-A3    | Reserved                                      |             |
| A4       | Safety Switch                                 | DF          |
| A5       | Actuator Override Switch                      | DF          |
| A6       | Actuator Power                                | OOC         |
| A7       | Start Delay                                   | DV          |
| A8       | Parameter  Block  Size                        | CL          |
| A9       | Device-Managed Pool                           | SF          |
| AA       | Shared Parameter Blocks                       | SF          |
| AB       | Create  New  Effect  Parameter  Block  Report | CL          |
| AC       | RAM Pool Available                            | DV          |
| AD-FFFF  | Reserved                                      |             |
