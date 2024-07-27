---
name: Sensors Page
alias: sen
code: 0x20
---
# Usage Table

| Usage ID  | Usage Name                                                     | Usage Types |
|-----------|----------------------------------------------------------------|-------------|
| 00        | Undefined                                                      |             |
| 01        | Sensor                                                         | CA/CP       |
| 02-0F     | Reserved                                                       |             |
| 10        | Biometric                                                      | CA/CP       |
| 11        | Biometric:  Human  Presence                                    | CA/CP       |
| 12        | Biometric:  Human  Proximity                                   | CA/CP       |
| 13        | Biometric:  Human  Touch                                       | CA/CP       |
| 14        | Biometric:  Blood  Pressure                                    | CA/CP       |
| 15        | Biometric:  Body  Temperature                                  | CA/CP       |
| 16        | Biometric:  Heart  Rate                                        | CA/CP       |
| 17        | Biometric:  Heart  Rate  Variability                           | CA/CP       |
| 18        | Biometric:  Peripheral  Oxygen  Saturation                     | CA/CP       |
| 19        | Biometric:  Respiratory  Rate                                  | CA/CP       |
| 1A-1F     | Reserved                                                       |             |
| 20        | Electrical                                                     | CA/CP       |
| 21        | Electrical:  Capacitance                                       | CA/CP       |
| 22        | Electrical:  Current                                           | CA/CP       |
| 23        | Electrical:  Power                                             | CA/CP       |
| 24        | Electrical:  Inductance                                        | CA/CP       |
| 25        | Electrical:  Resistance                                        | CA/CP       |
| 26        | Electrical:  Voltage                                           | CA/CP       |
| 27        | Electrical:  Potentiometer                                     | CA/CP       |
| 28        | Electrical:  Frequency                                         | CA/CP       |
| 29        | Electrical:  Period                                            | CA/CP       |
| 2A-2F     | Reserved                                                       |             |
| 30        | Environmental                                                  | CA/CP       |
| 31        | Environmental:  Atmospheric  Pressure                          | CA/CP       |
| 32        | Environmental:  Humidity                                       | CA/CP       |
| 33        | Environmental:  Temperature                                    | CA/CP       |
| 34        | Environmental:  Wind  Direction                                | CA/CP       |
| 35        | Environmental:  Wind  Speed                                    | CA/CP       |
| 36        | Environmental:  Air  Quality                                   | CA/CP       |
| 37        | Environmental:  Heat  Index                                    | CA/CP       |
| 38        | Environmental:  Surface  Temperature                           | CA/CP       |
| 39        | Environmental:  Volatile  Organic  Compounds                   | CA/CP       |
| 3A        | Environmental:  Object  Presence                               | CA/CP       |
| 3B        | Environmental:  Object  Proximity                              | CA/CP       |
| 3C-3F     | Reserved                                                       |             |
| 40        | Light                                                          | CA/CP       |
| 41        | Light:  Ambient  Light                                         | CA/CP       |
| 42        | Light:  Consumer  Infrared                                     | CA/CP       |
| 43        | Light:  Infrared  Light                                        | CA/CP       |
| 44        | Light:  Visible  Light                                         | CA/CP       |
| 45        | Light:  Ultraviolet  Light                                     | CA/CP       |
| 46-4F     | Reserved                                                       |             |
| 50        | Location                                                       | CA/CP       |
| 51        | Location:  Broadcast                                           | CA/CP       |
| 52        | Location:  Dead  Reckoning                                     | CA/CP       |
| 53        | Location:  GPS  (Global  Positioning  System)                  | CA/CP       |
| 54        | Location:  Lookup                                              | CA/CP       |
| 55        | Location:  Other                                               | CA/CP       |
| 56        | Location:  Static                                              | CA/CP       |
| 57        | Location:  Triangulation                                       | CA/CP       |
| 58-5F     | Reserved                                                       |             |
| 60        | Mechanical                                                     | CA/CP       |
| 61        | Mechanical:  Boolean  Switch                                   | CA/CP       |
| 62        | Mechanical:  Boolean  Switch  Array                            | CA/CP       |
| 63        | Mechanical:  Multivalue  Switch                                | CA/CP       |
| 64        | Mechanical:  Force                                             | CA/CP       |
| 65        | Mechanical:  Pressure                                          | CA/CP       |
| 66        | Mechanical:  Strain                                            | CA/CP       |
| 67        | Mechanical:  Weight                                            | CA/CP       |
| 68        | Mechanical:  Haptic  Vibrator                                  | CA/CP       |
| 69        | Mechanical:  Hall  Effect  Switch                              | CA/CP       |
| 6A-6F     | Reserved                                                       |             |
| 70        | Motion                                                         | CA/CP       |
| 71        | Motion:  Accelerometer  1D                                     | CA/CP       |
| 72        | Motion:  Accelerometer  2D                                     | CA/CP       |
| 73        | Motion:  Accelerometer  3D                                     | CA/CP       |
| 74        | Motion:  Gyrometer  1D                                         | CA/CP       |
| 75        | Motion:  Gyrometer  2D                                         | CA/CP       |
| 76        | Motion:  Gyrometer  3D                                         | CA/CP       |
| 77        | Motion:  Motion  Detector                                      | CA/CP       |
| 78        | Motion:  Speedometer                                           | CA/CP       |
| 79        | Motion:  Accelerometer                                         | CA/CP       |
| 7A        | Motion:  Gyrometer                                             | CA/CP       |
| 7B        | Motion:  Gravity  Vector                                       | CA/CP       |
| 7C        | Motion:  Linear  Accelerometer                                 | CA/CP       |
| 7D-7F     | Reserved                                                       |             |
| 80        | Orientation                                                    | CA/CP       |
| 81        | Orientation:  Compass  1D                                      | CA/CP       |
| 82        | Orientation:  Compass  2D                                      | CA/CP       |
| 83        | Orientation:  Compass  3D                                      | CA/CP       |
| 84        | Orientation:  Inclinometer  1D                                 | CA/CP       |
| 85        | Orientation:  Inclinometer  2D                                 | CA/CP       |
| 86        | Orientation:  Inclinometer  3D                                 | CA/CP       |
| 87        | Orientation:  Distance  1D                                     | CA/CP       |
| 88        | Orientation:  Distance  2D                                     | CA/CP       |
| 89        | Orientation:  Distance  3D                                     | CA/CP       |
| 8A        | Orientation:  Device  Orientation                              | CA/CP       |
| 8B        | Orientation:  Compass                                          | CA/CP       |
| 8C        | Orientation:  Inclinometer                                     | CA/CP       |
| 8D        | Orientation:  Distance                                         | CA/CP       |
| 8E        | Orientation:  Relative  Orientation                            | CA/CP       |
| 8F        | Orientation:  Simple  Orientation                              | CA/CP       |
| 90        | Scanner                                                        | CA/CP       |
| 91        | Scanner:  Barcode                                              | CA/CP       |
| 92        | Scanner:  RFID                                                 | CA/CP       |
| 93        | Scanner:  NFC                                                  | CA/CP       |
| 94-9F     | Reserved                                                       |             |
| A0        | Time                                                           | CA/CP       |
| A1        | Time:  Alarm  Timer                                            | CA/CP       |
| A2        | Time:  Real  Time  Clock                                       | CA/CP       |
| A3-AF     | Reserved                                                       |             |
| B0        | Personal  Activity                                             | CA/CP       |
| B1        | Personal  Activity:  Activity  Detection                       | CA/CP       |
| B2        | Personal  Activity:  Device  Position                          | CA/CP       |
| B3        | Personal  Activity:  Pedometer                                 | CA/CP       |
| B4        | Personal  Activity:  Step  Detection                           | CA/CP       |
| B5-BF     | Reserved                                                       |             |
| C0        | Orientation  Extended                                          | CA/CP       |
| C1        | Orientation  Extended:  Geomagnetic  Orientation               | CA/CP       |
| C2        | Orientation  Extended:  Magnetometer                           | CA/CP       |
| C3-CF     | Reserved                                                       |             |
| D0        | Gesture                                                        | CA/CP       |
| D1        | Gesture:  Chassis  Flip  Gesture                               | CA/CP       |
| D2        | Gesture:  Hinge  Fold  Gesture                                 | CA/CP       |
| D3-DF     | Reserved                                                       |             |
| E0        | Other                                                          | CA/CP       |
| E1        | Other:  Custom                                                 | CA/CP       |
| E2        | Other:  Generic                                                | CA/CP       |
| E3        | Other:  Generic  Enumerator                                    | CA/CP       |
| E4        | Other:  Hinge  Angle                                           | CA/CP       |
| E5-EF     | Reserved                                                       |             |
| F0        | Vendor  Reserved  1                                            | CA/CP       |
| F1        | Vendor  Reserved  2                                            | CA/CP       |
| F2        | Vendor  Reserved  3                                            | CA/CP       |
| F3        | Vendor  Reserved  4                                            | CA/CP       |
| F4        | Vendor  Reserved  5                                            | CA/CP       |
| F5        | Vendor  Reserved  6                                            | CA/CP       |
| F6        | Vendor  Reserved  7                                            | CA/CP       |
| F7        | Vendor  Reserved  8                                            | CA/CP       |
| F8        | Vendor  Reserved  9                                            | CA/CP       |
| F9        | Vendor  Reserved  10                                           | CA/CP       |
| FA        | Vendor  Reserved  11                                           | CA/CP       |
| FB        | Vendor  Reserved  12                                           | CA/CP       |
| FC        | Vendor  Reserved  13                                           | CA/CP       |
| FD        | Vendor  Reserved  14                                           | CA/CP       |
| FE        | Vendor  Reserved  15                                           | CA/CP       |
| FF        | Vendor  Reserved  16                                           | CA/CP       |
| 100-1FF   | Reserved                                                       |             |
| 200       | Event                                                          | DV          |
| 201       | Event:  Sensor  State                                          | NAry        |
| 202       | Event:  Sensor  Event                                          | NAry        |
| 203-2FF   | Reserved                                                       |             |
| 300       | Property                                                       | DV          |
| 301       | Property:  Friendly Name                                       | SV          |
| 302       | Property:  Persistent Unique ID                                | DV          |
| 303       | Property:  Sensor Status                                       | DV          |
| 304       | Property:  Minimum Report Interval                             | SV          |
| 305       | Property:  Sensor Manufacturer                                 | SV          |
| 306       | Property:  Sensor Model                                        | SV          |
| 307       | Property:  Sensor Serial Number                                | SV          |
| 308       | Property:  Sensor Description                                  | SV          |
| 309       | Property:  Sensor  Connection  Type                            | NAry        |
| 30A       | Property:  Sensor Device Path                                  | DV          |
| 30B       | Property:  Hardware Revision                                   | SV          |
| 30C       | Property:  Firmware Version                                    | SV          |
| 30D       | Property:  Release Date                                        | SV          |
| 30E       | Property:  Report Interval                                     | DV          |
| 30F       | Property:  Change Sensitivity Absolute                         | DV          |
| 310       | Property:  Change Sensitivity Percent of Range                 | DV          |
| 311       | Property:  Change Sensitivity Percent Relative                 | DV          |
| 312       | Property:  Accuracy                                            | DV          |
| 313       | Property:  Resolution                                          | DV          |
| 314       | Property:  Maximum                                             | DV          |
| 315       | Property:  Minimum                                             | DV          |
| 316       | Property:  Reporting  State                                    | NAry        |
| 317       | Property:  Sampling Rate                                       | DV          |
| 318       | Property:  Response Curve                                      | DV          |
| 319       | Property:  Power  State                                        | NAry        |
| 31A       | Property:  Maximum FIFO Events                                 | SV          |
| 31B       | Property:  Report Latency                                      | DV          |
| 31C       | Property:  Flush FIFO Events                                   | DF          |
| 31D       | Property:  Maximum Power Consumption                           | DV          |
| 31E       | Property:  Is Primary                                          | DF          |
| 31F       | Property:  Human  Presence  Detection  Type                    | NAry        |
| 320-3FF   | Reserved                                                       |             |
| 400       | Data Field:  Location                                          | DV          |
| 401-401   | Reserved                                                       |             |
| 402       | Data Field:  Altitude Antenna Sea Level                        | SV          |
| 403       | Data Field:  Differential Reference Station ID                 | SV          |
| 404       | Data Field:  Altitude Ellipsoid Error                          | SV          |
| 405       | Data Field:  Altitude Ellipsoid                                | SV          |
| 406       | Data Field:  Altitude Sea Level Error                          | SV          |
| 407       | Data Field:  Altitude Sea Level                                | SV          |
| 408       | Data Field:  Differential GPS Data Age                         | SV          |
| 409       | Data Field:  Error Radius                                      | SV          |
| 40A       | Data  Field:  Fix  Quality                                     | NAry        |
| 40B       | Data  Field:  Fix  Type                                        | NAry        |
| 40C       | Data Field:  Geoidal Separation                                | SV          |
| 40D       | Data  Field:  GPS  Operation  Mode                             | NAry        |
| 40E       | Data  Field:  GPS  Selection  Mode                             | NAry        |
| 40F       | Data  Field:  GPS  Status                                      | NAry        |
| 410       | Data Field:  Position Dilution of Precision                    | SV          |
| 411       | Data Field:  Horizontal Dilution of Precision                  | SV          |
| 412       | Data Field:  Vertical Dilution of Precision                    | SV          |
| 413       | Data Field:  Latitude                                          | SV          |
| 414       | Data Field:  Longitude                                         | SV          |
| 415       | Data Field:  True Heading                                      | SV          |
| 416       | Data Field:  Magnetic Heading                                  | SV          |
| 417       | Data Field:  Magnetic Variation                                | SV          |
| 418       | Data Field:  Speed                                             | SV          |
| 419       | Data Field:  Satellites in View                                | SV          |
| 41A       | Data Field:  Satellites in View Azimuth                        | SV          |
| 41B       | Data Field:  Satellites in View Elevation                      | SV          |
| 41C       | Data Field:  Satellites in View IDs                            | SV          |
| 41D       | Data Field:  Satellites in View PRNs                           | SV          |
| 41E       | Data Field:  Satellites in View S/N Ratios                     | SV          |
| 41F       | Data Field:  Satellites Used Count                             | SV          |
| 420       | Data Field:  Satellites Used PRNs                              | SV          |
| 421       | Data Field:  NMEA Sentence                                     | SV          |
| 422       | Data Field:  Address Line 1                                    | SV          |
| 423       | Data Field:  Address Line 2                                    | SV          |
| 424       | Data Field:  City                                              | SV          |
| 425       | Data Field:  State or Province                                 | SV          |
| 426       | Data Field:  Country or Region                                 | SV          |
| 427       | Data Field:  Postal Code                                       | SV          |
| 428-429   | Reserved                                                       |             |
| 42A       | Property:  Location                                            | DV          |
| 42B       | Property:  Location  Desired  Accuracy                         | NAry        |
| 42C-42F   | Reserved                                                       |             |
| 430       | Data Field:  Environmental                                     | SV          |
| 431       | Data Field:  Atmospheric Pressure                              | SV          |
| 432-432   | Reserved                                                       |             |
| 433       | Data Field:  Relative Humidity                                 | SV          |
| 434       | Data Field:  Temperature                                       | SV          |
| 435       | Data Field:  Wind Direction                                    | SV          |
| 436       | Data Field:  Wind Speed                                        | SV          |
| 437       | Data Field:  Air Quality Index                                 | SV          |
| 438       | Data Field:  Equivalent CO2                                    | SV          |
| 439       | Data Field:  Volatile Organic Compound Concentration           | SV          |
| 43A       | Data Field:  Object Presence                                   | SF          |
| 43B       | Data Field:  Object Proximity Range                            | SV          |
| 43C       | Data Field:  Object Proximity Out of Range                     | SF          |
| 43D-43F   | Reserved                                                       |             |
| 440       | Property:  Environmental                                       | SV          |
| 441       | Property:  Reference Pressure                                  | SV          |
| 442-44F   | Reserved                                                       |             |
| 450       | Data Field:  Motion                                            | DV          |
| 451       | Data Field:  Motion State                                      | SF          |
| 452       | Data Field:  Acceleration                                      | SV          |
| 453       | Data Field:  Acceleration Axis X                               | SV          |
| 454       | Data Field:  Acceleration Axis Y                               | SV          |
| 455       | Data Field:  Acceleration Axis Z                               | SV          |
| 456       | Data Field:  Angular Velocity                                  | SV          |
| 457       | Data Field:  Angular Velocity about X Axis                     | SV          |
| 458       | Data Field:  Angular Velocity about Y Axis                     | SV          |
| 459       | Data Field:  Angular Velocity about Z Axis                     | SV          |
| 45A       | Data Field:  Angular Position                                  | SV          |
| 45B       | Data Field:  Angular Position about X Axis                     | SV          |
| 45C       | Data Field:  Angular Position about Y Axis                     | SV          |
| 45D       | Data Field:  Angular Position about Z Axis                     | SV          |
| 45E       | Data Field:  Motion Speed                                      | SV          |
| 45F       | Data Field:  Motion Intensity                                  | SV          |
| 460-46F   | Reserved                                                       |             |
| 470       | Data Field:  Orientation                                       | DV          |
| 471       | Data Field:  Heading                                           | SV          |
| 472       | Data Field:  Heading X Axis                                    | SV          |
| 473       | Data Field:  Heading Y Axis                                    | SV          |
| 474       | Data Field:  Heading Z Axis                                    | SV          |
| 475       | Data Field:  Heading Compensated Magnetic North                | SV          |
| 476       | Data Field:  Heading Compensated True North                    | SV          |
| 477       | Data Field:  Heading Magnetic North                            | SV          |
| 478       | Data Field:  Heading True North                                | SV          |
| 479       | Data Field:  Distance                                          | SV          |
| 47A       | Data Field:  Distance X Axis                                   | SV          |
| 47B       | Data Field:  Distance Y Axis                                   | SV          |
| 47C       | Data Field:  Distance Z Axis                                   | SV          |
| 47D       | Data Field:  Distance Out-of-Range                             | SF          |
| 47E       | Data Field:  Tilt                                              | SV          |
| 47F       | Data Field:  Tilt X Axis                                       | SV          |
| 480       | Data Field:  Tilt Y Axis                                       | SV          |
| 481       | Data Field:  Tilt Z Axis                                       | SV          |
| 482       | Data Field:  Rotation Matrix                                   | SV          |
| 483       | Data Field:  Quaternion                                        | SV          |
| 484       | Data Field:  Magnetic Flux                                     | SV          |
| 485       | Data Field:  Magnetic Flux X Axis                              | SV          |
| 486       | Data Field:  Magnetic Flux Y Axis                              | SV          |
| 487       | Data Field:  Magnetic Flux Z Axis                              | SV          |
| 488       | Data  Field:  Magnetometer  Accuracy                           | NAry        |
| 489       | Data  Field:  Simple  Orientation  Direction                   | NAry        |
| 48A-48F   | Reserved                                                       |             |
| 490       | Data Field:  Mechanical                                        | DV          |
| 491       | Data Field:  Boolean Switch State                              | SF          |
| 492       | Data Field:  Boolean Switch Array States                       | SV          |
| 493       | Data Field:  Multivalue Switch Value                           | SV          |
| 494       | Data Field:  Force                                             | SV          |
| 495       | Data Field:  Absolute Pressure                                 | SV          |
| 496       | Data Field:  Gauge Pressure                                    | SV          |
| 497       | Data Field:  Strain                                            | SV          |
| 498       | Data Field:  Weight                                            | SV          |
| 499-49F   | Reserved                                                       |             |
| 4A0       | Property:  Mechanical                                          | DV          |
| 4A1       | Property:  Vibration State                                     | DF          |
| 4A2       | Property:  Forward Vibration Speed                             | DV          |
| 4A3       | Property:  Backward Vibration Speed                            | DV          |
| 4A4-4AF   | Reserved                                                       |             |
| 4B0       | Data Field:  Biometric                                         | DV          |
| 4B1       | Data Field:  Human Presence                                    | SF          |
| 4B2       | Data Field:  Human Proximity Range                             | SV          |
| 4B3       | Data Field:  Human Proximity Out of Range                      | SF          |
| 4B4       | Data Field:  Human Touch State                                 | SF          |
| 4B5       | Data Field:  Blood Pressure                                    | SV          |
| 4B6       | Data Field:  Blood Pressure Diastolic                          | SV          |
| 4B7       | Data Field:  Blood Pressure Systolic                           | SV          |
| 4B8       | Data Field:  Heart Rate                                        | SV          |
| 4B9       | Data Field:  Resting Heart Rate                                | SV          |
| 4BA       | Data Field:  Heartbeat Interval                                | SV          |
| 4BB       | Data Field:  Respiratory Rate                                  | SV          |
| 4BC       | Data Field:  SpO2                                              | SV          |
| 4BD       | Data Field:  Human Attention Detected                          | MC          |
| 4BE-4CF   | Reserved                                                       |             |
| 4D0       | Data Field:  Light                                             | DV          |
| 4D1       | Data Field:  Illuminance                                       | SV          |
| 4D2       | Data Field:  Color Temperature                                 | SV          |
| 4D3       | Data Field:  Chromaticity                                      | SV          |
| 4D4       | Data Field:  Chromaticity X                                    | SV          |
| 4D5       | Data Field:  Chromaticity Y                                    | SV          |
| 4D6       | Data Field:  Consumer IR Sentence Receive                      | SV          |
| 4D7       | Data Field:  Infrared Light                                    | SV          |
| 4D8       | Data Field:  Red Light                                         | SV          |
| 4D9       | Data Field:  Green Light                                       | SV          |
| 4DA       | Data Field:  Blue Light                                        | SV          |
| 4DB       | Data Field:  Ultraviolet A Light                               | SV          |
| 4DC       | Data Field:  Ultraviolet B Light                               | SV          |
| 4DD       | Data Field:  Ultraviolet Index                                 | SV          |
| 4DE       | Data Field:  Near Infrared Light                               | SV          |
| 4DF       | Property:  Light                                               | DV          |
| 4E+0      | Property:  Consumer IR Sentence Send                           | DV          |
| 4E1-4E1   | Reserved                                                       |             |
| 4E+2      | Property:  Auto Brightness Preferred                           | DF          |
| 4E+3      | Property:  Auto Color Preferred                                | DF          |
| 4E4-4EF   | Reserved                                                       |             |
| 4F0       | Data Field:  Scanner                                           | DV          |
| 4F1       | Data Field:  RFID Tag 40 Bit                                   | SV          |
| 4F2       | Data Field:  NFC Sentence Receive                              | SV          |
| 4F3-4F7   | Reserved                                                       |             |
| 4F8       | Property:  Scanner                                             | DV          |
| 4F9       | Property:  NFC Sentence Send                                   | SV          |
| 4FA-4FF   | Reserved                                                       |             |
| 500       | Data Field:  Electrical                                        | SV          |
| 501       | Data Field:  Capacitance                                       | SV          |
| 502       | Data Field:  Current                                           | SV          |
| 503       | Data Field:  Electrical Power                                  | SV          |
| 504       | Data Field:  Inductance                                        | SV          |
| 505       | Data Field:  Resistance                                        | SV          |
| 506       | Data Field:  Voltage                                           | SV          |
| 507       | Data Field:  Frequency                                         | SV          |
| 508       | Data Field:  Period                                            | SV          |
| 509       | Data Field:  Percent of Range                                  | SV          |
| 50A-51F   | Reserved                                                       |             |
| 520       | Data Field:  Time                                              | DV          |
| 521       | Data Field:  Year                                              | SV          |
| 522       | Data Field:  Month                                             | SV          |
| 523       | Data Field:  Day                                               | SV          |
| 524       | Data  Field:  Day  of  Week                                    | NAry        |
| 525       | Data Field:  Hour                                              | SV          |
| 526       | Data Field:  Minute                                            | SV          |
| 527       | Data Field:  Second                                            | SV          |
| 528       | Data Field:  Millisecond                                       | SV          |
| 529       | Data Field:  Timestamp                                         | SV          |
| 52A       | Data Field:  Julian Day of Year                                | SV          |
| 52B       | Data Field:  Time Since System Boot                            | SV          |
| 52C-52F   | Reserved                                                       |             |
| 530       | Property:  Time                                                | DV          |
| 531       | Property:  Time Zone Offset from UTC                           | DV          |
| 532       | Property:  Time Zone Name                                      | DV          |
| 533       | Property:  Daylight Savings Time Observed                      | DF          |
| 534       | Property:  Time Trim Adjustment                                | DV          |
| 535       | Property:  Arm Alarm                                           | DF          |
| 536-53F   | Reserved                                                       |             |
| 540       | Data Field:  Custom                                            | DV          |
| 541       | Data Field:  Custom Usage                                      | SV          |
| 542       | Data Field:  Custom Boolean Array                              | SV          |
| 543       | Data Field:  Custom Value                                      | SV          |
| 544       | Data Field:  Custom Value 1                                    | SV          |
| 545       | Data Field:  Custom Value 2                                    | SV          |
| 546       | Data Field:  Custom Value 3                                    | SV          |
| 547       | Data Field:  Custom Value 4                                    | SV          |
| 548       | Data Field:  Custom Value 5                                    | SV          |
| 549       | Data Field:  Custom Value 6                                    | SV          |
| 54A       | Data Field:  Custom Value 7                                    | SV          |
| 54B       | Data Field:  Custom Value 8                                    | SV          |
| 54C       | Data Field:  Custom Value 9                                    | SV          |
| 54D       | Data Field:  Custom Value 10                                   | SV          |
| 54E       | Data Field:  Custom Value 11                                   | SV          |
| 54F       | Data Field:  Custom Value 12                                   | SV          |
| 550       | Data Field:  Custom Value 13                                   | SV          |
| 551       | Data Field:  Custom Value 14                                   | SV          |
| 552       | Data Field:  Custom Value 15                                   | SV          |
| 553       | Data Field:  Custom Value 16                                   | SV          |
| 554       | Data Field:  Custom Value 17                                   | SV          |
| 555       | Data Field:  Custom Value 18                                   | SV          |
| 556       | Data Field:  Custom Value 19                                   | SV          |
| 557       | Data Field:  Custom Value 20                                   | SV          |
| 558       | Data Field:  Custom Value 21                                   | SV          |
| 559       | Data Field:  Custom Value 22                                   | SV          |
| 55A       | Data Field:  Custom Value 23                                   | SV          |
| 55B       | Data Field:  Custom Value 24                                   | SV          |
| 55C       | Data Field:  Custom Value 25                                   | SV          |
| 55D       | Data Field:  Custom Value 26                                   | SV          |
| 55E       | Data Field:  Custom Value 27                                   | SV          |
| 55F       | Data Field:  Custom Value 28                                   | SV          |
| 560       | Data Field:  Generic                                           | DV          |
| 561       | Data Field:  Generic GUID or PROPERTYKEY                       | SV          |
| 562       | Data Field:  Generic Category GUID                             | SV          |
| 563       | Data Field:  Generic Type GUID                                 | SV          |
| 564       | Data Field:  Generic Event PROPERTYKEY                         | SV          |
| 565       | Data Field:  Generic Property PROPERTYKEY                      | SV          |
| 566       | Data Field:  Generic Data Field PROPERTYKEY                    | SV          |
| 567       | Data Field:  Generic Event                                     | SV          |
| 568       | Data Field:  Generic Property                                  | SV          |
| 569       | Data Field:  Generic Data Field                                | SV          |
| 56A       | Data Field:  Enumerator Table Row Index                        | SV          |
| 56B       | Data Field:  Enumerator Table Row Count                        | SV          |
| 56C       | Data  Field:  Generic  GUID  or  PROPERTYKEY  kind             | NAry        |
| 56D       | Data Field:  Generic GUID                                      | SV          |
| 56E       | Data Field:  Generic PROPERTYKEY                               | SV          |
| 56F       | Data Field:  Generic Top Level Collection ID                   | SV          |
| 570       | Data Field:  Generic Report ID                                 | SV          |
| 571       | Data Field:  Generic Report Item Position Index                | SV          |
| 572       | Data  Field:  Generic  Firmware  VARTYPE                       | NAry        |
| 573       | Data  Field:  Generic  Unit  of  Measure                       | NAry        |
| 574       | Data  Field:  Generic  Unit  Exponent                          | NAry        |
| 575       | Data Field:  Generic Report Size                               | SV          |
| 576       | Data Field:  Generic Report Count                              | SV          |
| 577-57F   | Reserved                                                       |             |
| 580       | Property:  Generic                                             | DV          |
| 581       | Property:  Enumerator Table Row Index                          | DV          |
| 582       | Property:  Enumerator Table Row Count                          | SV          |
| 583-58F   | Reserved                                                       |             |
| 590       | Data Field:  Personal Activity                                 | DV          |
| 591       | Data  Field:  Activity  Type                                   | NAry        |
| 592       | Data  Field:  Activity  State                                  | NAry        |
| 593       | Data  Field:  Device  Position                                 | NAry        |
| 594       | Data Field:  Step Count                                        | SV          |
| 595       | Data Field:  Step Count Reset                                  | DF          |
| 596       | Data Field:  Step Duration                                     | SV          |
| 597       | Data  Field:  Step  Type                                       | NAry        |
| 598-59F   | Reserved                                                       |             |
| 5A0       | Property:  Minimum Activity Detection Interval                 | DV          |
| 5A1       | Property:  Supported  Activity  Types                          | NAry        |
| 5A2       | Property:  Subscribed  Activity  Types                         | NAry        |
| 5A3       | Property:  Supported  Step  Types                              | NAry        |
| 5A4       | Property:  Subscribed  Step  Types                             | NAry        |
| 5A5       | Property:  Floor Height                                        | DV          |
| 5A6-5AF   | Reserved                                                       |             |
| 5B0       | Data Field:  Custom Type ID                                    | SV          |
| 5B1-5BF   | Reserved                                                       |             |
| 5C0       | Property:  Custom                                              | DV          |
| 5C1       | Property:  Custom Value 1                                      | DV          |
| 5C2       | Property:  Custom Value 2                                      | DV          |
| 5C3       | Property:  Custom Value 3                                      | DV          |
| 5C4       | Property:  Custom Value 4                                      | DV          |
| 5C5       | Property:  Custom Value 5                                      | DV          |
| 5C6       | Property:  Custom Value 6                                      | DV          |
| 5C7       | Property:  Custom Value 7                                      | DV          |
| 5C8       | Property:  Custom Value 8                                      | DV          |
| 5C9       | Property:  Custom Value 9                                      | DV          |
| 5CA       | Property:  Custom Value 10                                     | DV          |
| 5CB       | Property:  Custom Value 11                                     | DV          |
| 5CC       | Property:  Custom Value 12                                     | DV          |
| 5CD       | Property:  Custom Value 13                                     | DV          |
| 5CE       | Property:  Custom Value 14                                     | DV          |
| 5CF       | Property:  Custom Value 15                                     | DV          |
| 5D0       | Property:  Custom Value 16                                     | DV          |
| 5D1-5DF   | Reserved                                                       |             |
| 5E+0      | Data Field:  Hinge                                             | SV/DV       |
| 5E+1      | Data Field:  Hinge Angle                                       | SV/DV       |
| 5E2-5EF   | Reserved                                                       |             |
| 5F0       | Data Field:  Gesture Sensor                                    | DV          |
| 5F1       | Data  Field:  Gesture  State                                   | NAry        |
| 5F2       | Data Field:  Hinge Fold Initial Angle                          | SV          |
| 5F3       | Data Field:  Hinge Fold Final Angle                            | SV          |
| 5F4       | Data  Field:  Hinge  Fold  Contributing  Panel                 | NAry        |
| 5F5       | Data  Field:  Hinge  Fold  Type                                | NAry        |
| 5F6-7FF   | Reserved                                                       |             |
| 800       | Sensor State:  Undefined                                       | Sel         |
| 801       | Sensor State:  Ready                                           | Sel         |
| 802       | Sensor State:  Not Available                                   | Sel         |
| 803       | Sensor State:  No Data                                         | Sel         |
| 804       | Sensor State:  Initializing                                    | Sel         |
| 805       | Sensor State:  Access Denied                                   | Sel         |
| 806       | Sensor State:  Error                                           | Sel         |
| 807-80F   | Reserved                                                       |             |
| 810       | Sensor Event:  Unknown                                         | Sel         |
| 811       | Sensor Event:  State Changed                                   | Sel         |
| 812       | Sensor Event:  Property Changed                                | Sel         |
| 813       | Sensor Event:  Data Updated                                    | Sel         |
| 814       | Sensor Event:  Poll Response                                   | Sel         |
| 815       | Sensor Event:  Change Sensitivity                              | Sel         |
| 816       | Sensor Event:  Range Maximum Reached                           | Sel         |
| 817       | Sensor Event:  Range Minimum Reached                           | Sel         |
| 818       | Sensor Event:  High Threshold Cross Upward                     | Sel         |
| 819       | Sensor Event:  High Threshold Cross Downward                   | Sel         |
| 81A       | Sensor Event:  Low Threshold Cross Upward                      | Sel         |
| 81B       | Sensor Event:  Low Threshold Cross Downward                    | Sel         |
| 81C       | Sensor Event:  Zero Threshold Cross Upward                     | Sel         |
| 81D       | Sensor Event:  Zero Threshold Cross Downward                   | Sel         |
| 81E       | Sensor Event:  Period Exceeded                                 | Sel         |
| 81F       | Sensor Event:  Frequency Exceeded                              | Sel         |
| 820       | Sensor Event:  Complex Trigger                                 | Sel         |
| 821-82F   | Reserved                                                       |             |
| 830       | Connection Type:  PC Integrated                                | Sel         |
| 831       | Connection Type:  PC Attached                                  | Sel         |
| 832       | Connection Type:  PC External                                  | Sel         |
| 833-83F   | Reserved                                                       |             |
| 840       | Reporting State:  Report No Events                             | Sel         |
| 841       | Reporting State:  Report All Events                            | Sel         |
| 842       | Reporting State:  Report Threshold Events                      | Sel         |
| 843       | Reporting State:  Wake On No Events                            | Sel         |
| 844       | Reporting State:  Wake On All Events                           | Sel         |
| 845       | Reporting State:  Wake On Threshold Events                     | Sel         |
| 846-84F   | Reserved                                                       |             |
| 850       | Power State:  Undefined                                        | Sel         |
| 851       | Power State:  D0 Full Power                                    | Sel         |
| 852       | Power State:  D1 Low Power                                     | Sel         |
| 853       | Power State:  D2 Standby Power with Wakeup                     | Sel         |
| 854       | Power State:  D3 Sleep with Wakeup                             | Sel         |
| 855       | Power State:  D4 Power Off                                     | Sel         |
| 856-85F   | Reserved                                                       |             |
| 860       | Accuracy:  Default                                             | Sel         |
| 861       | Accuracy:  High                                                | Sel         |
| 862       | Accuracy:  Medium                                              | Sel         |
| 863       | Accuracy:  Low                                                 | Sel         |
| 864-86F   | Reserved                                                       |             |
| 870       | Fix Quality:  No Fix                                           | Sel         |
| 871       | Fix Quality:  GPS                                              | Sel         |
| 872       | Fix Quality:  DGPS                                             | Sel         |
| 873-87F   | Reserved                                                       |             |
| 880       | Fix Type:  No Fix                                              | Sel         |
| 881       | Fix Type:  GPS SPS Mode, Fix Valid                             | Sel         |
| 882       | Fix Type:  DGPS SPS Mode, Fix Valid                            | Sel         |
| 883       | Fix Type:  GPS PPS Mode, Fix Valid                             | Sel         |
| 884       | Fix Type:  Real Time Kinematic                                 | Sel         |
| 885       | Fix Type:  Float RTK                                           | Sel         |
| 886       | Fix Type:  Estimated (dead reckoned)                           | Sel         |
| 887       | Fix Type:  Manual Input Mode                                   | Sel         |
| 888       | Fix Type:  Simulator Mode                                      | Sel         |
| 889-88F   | Reserved                                                       |             |
| 890       | GPS Operation Mode:  Manual                                    | Sel         |
| 891       | GPS Operation Mode:  Automatic                                 | Sel         |
| 892-89F   | Reserved                                                       |             |
| 8A0       | GPS Selection Mode:  Autonomous                                | Sel         |
| 8A1       | GPS Selection Mode:  DGPS                                      | Sel         |
| 8A2       | GPS Selection Mode:  Estimated (dead reckoned)                 | Sel         |
| 8A3       | GPS Selection Mode:  Manual Input                              | Sel         |
| 8A4       | GPS Selection Mode:  Simulator                                 | Sel         |
| 8A5       | GPS Selection Mode:  Data Not Valid                            | Sel         |
| 8A6-8AF   | Reserved                                                       |             |
| 8B0       | GPS Status Data:  Valid                                        | Sel         |
| 8B1       | GPS Status Data:  Not Valid                                    | Sel         |
| 8B2-8BF   | Reserved                                                       |             |
| 8C0       | Day of Week:  Sunday                                           | Sel         |
| 8C1       | Day of Week:  Monday                                           | Sel         |
| 8C2       | Day of Week:  Tuesday                                          | Sel         |
| 8C3       | Day of Week:  Wednesday                                        | Sel         |
| 8C4       | Day of Week:  Thursday                                         | Sel         |
| 8C5       | Day of Week:  Friday                                           | Sel         |
| 8C6       | Day of Week:  Saturday                                         | Sel         |
| 8C7-8CF   | Reserved                                                       |             |
| 8D0       | Kind:  Category                                                | Sel         |
| 8D1       | Kind:  Type                                                    | Sel         |
| 8D2       | Kind:  Event                                                   | Sel         |
| 8D3       | Kind:  Property                                                | Sel         |
| 8D4       | Kind:  Data Field                                              | Sel         |
| 8D5-8DF   | Reserved                                                       |             |
| 8E+0      | Magnetometer Accuracy:  Low                                    | Sel         |
| 8E+1      | Magnetometer Accuracy:  Medium                                 | Sel         |
| 8E+2      | Magnetometer Accuracy:  High                                   | Sel         |
| 8E3-8EF   | Reserved                                                       |             |
| 8F0       | Simple Orientation Direction:  Not Rotated                     | Sel         |
| 8F1       | Simple Orientation Direction:  Rotated 90 Degrees CCW          | Sel         |
| 8F2       | Simple Orientation Direction:  Rotated 180 Degrees CCW         | Sel         |
| 8F3       | Simple Orientation Direction:  Rotated 270 Degrees CCW         | Sel         |
| 8F4       | Simple Orientation Direction:  Face Up                         | Sel         |
| 8F5       | Simple Orientation Direction:  Face Down                       | Sel         |
| 8F6-8FF   | Reserved                                                       |             |
| 900       | VT_NULL                                                        | Sel         |
| 901       | VT_BOOL                                                        | Sel         |
| 902       | VT_UI1                                                         | Sel         |
| 903       | VT_I1                                                          | Sel         |
| 904       | VT_UI2                                                         | Sel         |
| 905       | VT_I2                                                          | Sel         |
| 906       | VT_UI4                                                         | Sel         |
| 907       | VT_I4                                                          | Sel         |
| 908       | VT_UI8                                                         | Sel         |
| 909       | VT_I8                                                          | Sel         |
| 90A       | VT_R4                                                          | Sel         |
| 90B       | VT_R8                                                          | Sel         |
| 90C       | VT_WSTR                                                        | Sel         |
| 90D       | VT_STR                                                         | Sel         |
| 90E       | VT_CLSID                                                       | Sel         |
| 90F       | VT_VECTOR VT_UI1                                               | Sel         |
| 910       | VT_F16E0                                                       | Sel         |
| 911       | VT_F16E1                                                       | Sel         |
| 912       | VT_F16E2                                                       | Sel         |
| 913       | VT_F16E3                                                       | Sel         |
| 914       | VT_F16E4                                                       | Sel         |
| 915       | VT_F16E5                                                       | Sel         |
| 916       | VT_F16E6                                                       | Sel         |
| 917       | VT_F16E7                                                       | Sel         |
| 918       | VT_F16E8                                                       | Sel         |
| 919       | VT_F16E9                                                       | Sel         |
| 91A       | VT_F16EA                                                       | Sel         |
| 91B       | VT_F16EB                                                       | Sel         |
| 91C       | VT_F16EC                                                       | Sel         |
| 91D       | VT_F16ED                                                       | Sel         |
| 91E       | VT_F16EE                                                       | Sel         |
| 91F       | VT_F16EF                                                       | Sel         |
| 920       | VT_F32E0                                                       | Sel         |
| 921       | VT_F32E1                                                       | Sel         |
| 922       | VT_F32E2                                                       | Sel         |
| 923       | VT_F32E3                                                       | Sel         |
| 924       | VT_F32E4                                                       | Sel         |
| 925       | VT_F32E5                                                       | Sel         |
| 926       | VT_F32E6                                                       | Sel         |
| 927       | VT_F32E7                                                       | Sel         |
| 928       | VT_F32E8                                                       | Sel         |
| 929       | VT_F32E9                                                       | Sel         |
| 92A       | VT_F32EA                                                       | Sel         |
| 92B       | VT_F32EB                                                       | Sel         |
| 92C       | VT_F32EC                                                       | Sel         |
| 92D       | VT_F32ED                                                       | Sel         |
| 92E       | VT_F32EE                                                       | Sel         |
| 92F       | VT_F32EF                                                       | Sel         |
| 930       | Activity Type:  Unknown                                        | Sel         |
| 931       | Activity Type:  Stationary                                     | Sel         |
| 932       | Activity Type:  Fidgeting                                      | Sel         |
| 933       | Activity Type:  Walking                                        | Sel         |
| 934       | Activity Type:  Running                                        | Sel         |
| 935       | Activity Type:  In Vehicle                                     | Sel         |
| 936       | Activity Type:  Biking                                         | Sel         |
| 937       | Activity Type:  Idle                                           | Sel         |
| 938-93F   | Reserved                                                       |             |
| 940       | Unit:  Not Specified                                           | Sel         |
| 941       | Unit:  Lux                                                     | Sel         |
| 942       | Unit:  Degrees Kelvin                                          | Sel         |
| 943       | Unit:  Degrees Celsius                                         | Sel         |
| 944       | Unit:  Pascal                                                  | Sel         |
| 945       | Unit:  Newton                                                  | Sel         |
| 946       | Unit:  Meters/Second                                           | Sel         |
| 947       | Unit:  Kilogram                                                | Sel         |
| 948       | Unit:  Meter                                                   | Sel         |
| 949       | Unit:  Meters/Second/Second                                    | Sel         |
| 94A       | Unit:  Farad                                                   | Sel         |
| 94B       | Unit:  Ampere                                                  | Sel         |
| 94C       | Unit:  Watt                                                    | Sel         |
| 94D       | Unit:  Henry                                                   | Sel         |
| 94E       | Unit:  Ohm                                                     | Sel         |
| 94F       | Unit:  Volt                                                    | Sel         |
| 950       | Unit:  Hertz                                                   | Sel         |
| 951       | Unit:  Bar                                                     | Sel         |
| 952       | Unit:  Degrees Anti-clockwise                                  | Sel         |
| 953       | Unit:  Degrees Clockwise                                       | Sel         |
| 954       | Unit:  Degrees                                                 | Sel         |
| 955       | Unit:  Degrees/Second                                          | Sel         |
| 956       | Unit:  Degrees/Second/Second                                   | Sel         |
| 957       | Unit:  Knot                                                    | Sel         |
| 958       | Unit:  Percent                                                 | Sel         |
| 959       | Unit:  Second                                                  | Sel         |
| 95A       | Unit:  Millisecond                                             | Sel         |
| 95B       | Unit:  G                                                       | Sel         |
| 95C       | Unit:  Bytes                                                   | Sel         |
| 95D       | Unit:  Milligauss                                              | Sel         |
| 95E       | Unit:  Bits                                                    | Sel         |
| 95F-95F   | Reserved                                                       |             |
| 960       | Activity State:  No State Change                               | Sel         |
| 961       | Activity State:  Start Activity                                | Sel         |
| 962       | Activity State:  End Activity                                  | Sel         |
| 963-96F   | Reserved                                                       |             |
| 970       | Exponent 0                                                     | Sel         |
| 971       | Exponent 1                                                     | Sel         |
| 972       | Exponent 2                                                     | Sel         |
| 973       | Exponent 3                                                     | Sel         |
| 974       | Exponent 4                                                     | Sel         |
| 975       | Exponent 5                                                     | Sel         |
| 976       | Exponent 6                                                     | Sel         |
| 977       | Exponent 7                                                     | Sel         |
| 978       | Exponent 8                                                     | Sel         |
| 979       | Exponent 9                                                     | Sel         |
| 97A       | Exponent A                                                     | Sel         |
| 97B       | Exponent B                                                     | Sel         |
| 97C       | Exponent C                                                     | Sel         |
| 97D       | Exponent D                                                     | Sel         |
| 97E       | Exponent E                                                     | Sel         |
| 97F       | Exponent F                                                     | Sel         |
| 980       | Device Position:  Unknown                                      | Sel         |
| 981       | Device Position:  Unchanged                                    | Sel         |
| 982       | Device Position:  On Desk                                      | Sel         |
| 983       | Device Position:  In Hand                                      | Sel         |
| 984       | Device Position:  Moving in Bag                                | Sel         |
| 985       | Device Position:  Stationary in Bag                            | Sel         |
| 986-98F   | Reserved                                                       |             |
| 990       | Step Type:  Unknown                                            | Sel         |
| 991       | Step Type:  Running                                            | Sel         |
| 992       | Step Type:  Walking                                            | Sel         |
| 993-99F   | Reserved                                                       |             |
| 9A0       | Gesture State:  Unknown                                        | Sel         |
| 9A1       | Gesture State:  Started                                        | Sel         |
| 9A2       | Gesture State:  Completed                                      | Sel         |
| 9A3       | Gesture State:  Cancelled                                      | Sel         |
| 9A4-9AF   | Reserved                                                       |             |
| 9B0       | Hinge Fold Contributing Panel:  Unknown                        | Sel         |
| 9B1       | Hinge Fold Contributing Panel:  Panel 1                        | Sel         |
| 9B2       | Hinge Fold Contributing Panel:  Panel 2                        | Sel         |
| 9B3       | Hinge Fold Contributing Panel:  Both                           | Sel         |
| 9B4       | Hinge Fold Type:  Unknown                                      | Sel         |
| 9B5       | Hinge Fold Type:  Increasing                                   | Sel         |
| 9B6       | Hinge Fold Type:  Decreasing                                   | Sel         |
| 9B7-9BF   | Reserved                                                       |             |
| 9C0       | Human Presence Detection Type:  Vendor-Defined Non-Biometric   | Sel         |
| 9C1       | Human Presence Detection Type:  Vendor-Defined Biometric       | Sel         |
| 9C2       | Human Presence Detection Type:  Facial Biometric               | Sel         |
| 9C3       | Human Presence Detection Type:  Audio Biometric                | Sel         |
| 9C4-FFF   | Reserved                                                       |             |
| 1000      | Modifier:  Change Sensitivity Absolute                         | US          |
| 1001-10FF | Reserved                                                       |             |
| 1100-17FF | Reserved for use as Change Sensitivity Absolute modifier range |             |
| 1800-1FFF | Reserved                                                       |             |
| 2000      | Modifier:  Maximum                                             | US          |
| 2001-20FF | Reserved                                                       |             |
| 2100-27FF | Reserved for use as Maximum modifier range                     |             |
| 2800-2FFF | Reserved                                                       |             |
| 3000      | Modifier:  Minimum                                             | US          |
| 3001-30FF | Reserved                                                       |             |
| 3100-37FF | Reserved for use as Minimum modifier range                     |             |
| 3800-3FFF | Reserved                                                       |             |
| 4000      | Modifier:  Accuracy                                            | US          |
| 4001-40FF | Reserved                                                       |             |
| 4100-47FF | Reserved for use as Accuracy modifier range                    |             |
| 4800-4FFF | Reserved                                                       |             |
| 5000      | Modifier:  Resolution                                          | US          |
| 5001-50FF | Reserved                                                       |             |
| 5100-57FF | Reserved for use as Resolution modifier range                  |             |
| 5800-5FFF | Reserved                                                       |             |
| 6000      | Modifier:  Threshold High                                      | US          |
| 6001-60FF | Reserved                                                       |             |
| 6100-67FF | Reserved for use as Threshold High modifier range              |             |
| 6800-6FFF | Reserved                                                       |             |
| 7000      | Modifier:  Threshold Low                                       | US          |
| 7001-70FF | Reserved                                                       |             |
| 7100-77FF | Reserved for use as Threshold Low modifier range               |             |
| 7800-7FFF | Reserved                                                       |             |
| 8000      | Modifier:  Calibration Offset                                  | US          |
| 8001-80FF | Reserved                                                       |             |
| 8100-87FF | Reserved for use as Calibration Offset modifier range          |             |
| 8800-8FFF | Reserved                                                       |             |
| 9000      | Modifier:  Calibration Multiplier                              | US          |
| 9001-90FF | Reserved                                                       |             |
| 9100-97FF | Reserved for use as Calibration Multiplier modifier range      |             |
| 9800-9FFF | Reserved                                                       |             |
| A000      | Modifier:  Report Interval                                     | US          |
| A001-A0FF | Reserved                                                       |             |
| A100-A7FF | Reserved for use as Report Interval modifier range             |             |
| A800-AFFF | Reserved                                                       |             |
| B000      | Modifier:  Frequency Max                                       | US          |
| B001-B0FF | Reserved                                                       |             |
| B100-B7FF | Reserved for use as Frequency Max modifier range               |             |
| B800-BFFF | Reserved                                                       |             |
| C000      | Modifier:  Period Max                                          | US          |
| C001-C0FF | Reserved                                                       |             |
| C100-C7FF | Reserved for use as Period Max modifier range                  |             |
| C800-CFFF | Reserved                                                       |             |
| D000      | Modifier:  Change Sensitivity Percent of Range                 | US          |
| D001-D0FF | Reserved                                                       |             |
| D100-D7FF | Reserved for use as Change Sensitivity Percent modifier range  |             |
| D800-DFFF | Reserved                                                       |             |
| E000      | Modifier:  Change Sensitivity Percent Relative                 | US          |
| E001-E0FF | Reserved                                                       |             |
| E100-E7FF | Reserved for use as Change Sensitivity Percent modifier range  |             |
| E800-EFFF | Reserved                                                       |             |
| F000      | Modifier:  Vendor Reserved                                     | US          |
| F001-F0FF | Reserved                                                       |             |
| F100-F7FF | Reserved for use as Vendor Reserved modifier range             |             |
| F800-FFFF | Reserved                                                       |             |
