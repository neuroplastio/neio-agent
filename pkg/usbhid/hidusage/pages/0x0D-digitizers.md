| Usage ID | Usage Name                                     | Usage Types |
|----------|------------------------------------------------|-------------|
| 00       | Undefined                                      |             |
| 01       | Digitizer                                      | CA          |
| 02       | Pen                                            | CA          |
| 03       | Light  Pen                                     | CA          |
| 04       | Touch  Screen                                  | CA          |
| 05       | Touch  Pad                                     | CA          |
| 06       | Whiteboard                                     | CA          |
| 07       | Coordinate  Measuring  Machine                 | CA          |
| 08       | 3D  Digitizer                                  | CA          |
| 09       | Stereo  Plotter                                | CA          |
| 0A       | Articulated  Arm                               | CA          |
| 0B       | Armature                                       | CA          |
| 0C       | Multiple  Point  Digitizer                     | CA          |
| 0D       | Free  Space  Wand                              | CA          |
| 0E       | Device  Configuration                          | CA          |
| 0F       | Capacitive  Heat  Map  Digitizer               | CA          |
| 10-1F    | Reserved                                       |             |
| 20       | Stylus                                         | CA/CL       |
| 21       | Puck                                           | CL          |
| 22       | Finger                                         | CL          |
| 23       | Device  settings                               | CL          |
| 24       | Character  Gesture                             | CL          |
| 25-2F    | Reserved                                       |             |
| 30       | Tip Pressure                                   | DV          |
| 31       | Barrel Pressure                                | DV          |
| 32       | In Range                                       | MC          |
| 33       | Touch                                          | MC          |
| 34       | Untouch                                        | OSC         |
| 35       | Tap                                            | OSC         |
| 36       | Quality                                        | DV          |
| 37       | Data Valid                                     | MC          |
| 38       | Transducer Index                               | DV          |
| 39       | Tablet  Function  Keys                         | CL          |
| 3A       | Program  Change  Keys                          | CL          |
| 3B       | Battery Strength                               | DV          |
| 3C       | Invert                                         | MC          |
| 3D       | X Tilt                                         | DV          |
| 3E       | Y Tilt                                         | DV          |
| 3F       | Azimuth                                        | DV          |
| 40       | Altitude                                       | DV          |
| 41       | Twist                                          | DV          |
| 42       | Tip Switch                                     | MC          |
| 43       | Secondary Tip Switch                           | MC          |
| 44       | Barrel Switch                                  | MC          |
| 45       | Eraser                                         | MC          |
| 46       | Tablet Pick                                    | MC          |
| 47       | Touch Valid                                    | MC          |
| 48       | Width                                          | DV          |
| 49       | Height                                         | DV          |
| 4A-50    | Reserved                                       |             |
| 51       | Contact Identifier                             | DV          |
| 52       | Device Mode                                    | DV          |
| 53       | Device Identifier                              | DV/SV       |
| 54       | Contact Count                                  | DV          |
| 55       | Contact Count Maximum                          | SV          |
| 56       | Scan Time                                      | DV          |
| 57       | Surface Switch                                 | DF          |
| 58       | Button Switch                                  | DF          |
| 59       | Pad Type                                       | SF          |
| 5A       | Secondary Barrel Switch                        | MC          |
| 5B       | Transducer Serial Number                       | SV          |
| 5C       | Preferred Color                                | DV          |
| 5D       | Preferred Color is Locked                      | MC          |
| 5E       | Preferred Line Width                           | DV          |
| 5F       | Preferred Line Width is Locked                 | MC          |
| 60       | Latency Mode                                   | DF          |
| 61       | Gesture Character Quality                      | DV          |
| 62       | Character Gesture Data Length                  | DV          |
| 63       | Character Gesture Data                         | DV          |
| 64       | Gesture  Character  Encoding                   | NAry        |
| 65       | UTF8 Character Gesture Encoding                | Sel         |
| 66       | UTF16 Little Endian Character Gesture Encoding | Sel         |
| 67       | UTF16 Big Endian Character Gesture Encoding    | Sel         |
| 68       | UTF32 Little Endian Character Gesture Encoding | Sel         |
| 69       | UTF32 Big Endian Character Gesture Encoding    | Sel         |
| 6A       | Capacitive Heat Map Protocol Vendor ID         | SV          |
| 6B       | Capacitive Heat Map Protocol Version           | SV          |
| 6C       | Capacitive Heat Map Frame Data                 | DV          |
| 6D       | Gesture Character Enable                       | DF          |
| 6E       | Transducer Serial Number Part 2                | SV          |
| 6F       | No Preferred Color                             | DF          |
| 70       | Preferred  Line  Style                         | NAry        |
| 71       | Preferred Line Style is Locked                 | MC          |
| 72       | Ink                                            | Sel         |
| 73       | Pencil                                         | Sel         |
| 74       | Highlighter                                    | Sel         |
| 75       | Chisel Marker                                  | Sel         |
| 76       | Brush                                          | Sel         |
| 77       | No Preference                                  | Sel         |
| 78-7F    | Reserved                                       |             |
| 80       | Digitizer  Diagnostic                          | CL          |
| 81       | Digitizer  Error                               | NAry        |
| 82       | Err Normal Status                              | Sel         |
| 83       | Err Transducers Exceeded                       | Sel         |
| 84       | Err Full Trans Features Unavailable            | Sel         |
| 85       | Err Charge Low                                 | Sel         |
| 86-8F    | Reserved                                       |             |
| 90       | Transducer  Software  Info                     | CL          |
| 91       | Transducer Vendor Id                           | SV          |
| 92       | Transducer Product Id                          | SV          |
| 93       | Device  Supported  Protocols                   | NAry/CL     |
| 94       | Transducer  Supported  Protocols               | NAry/CL     |
| 95       | No Protocol                                    | Sel         |
| 96       | Wacom AES Protocol                             | Sel         |
| 97       | USI Protocol                                   | Sel         |
| 98       | Microsoft Pen Protocol                         | Sel         |
| 99-9F    | Reserved                                       |             |
| A0       | Supported  Report  Rates                       | SV/CL       |
| A1       | Report Rate                                    | DV          |
| A2       | Transducer Connected                           | SF          |
| A3       | Switch Disabled                                | Sel         |
| A4       | Switch Unimplemented                           | Sel         |
| A5       | Transducer  Switches                           | CL          |
| A6       | Transducer Index Selector                      | DV          |
| A7-AF    | Reserved                                       |             |
| B0       | Button Press Threshold                         | DV          |
| B1-FFFF  | Reserved                                       |             |
