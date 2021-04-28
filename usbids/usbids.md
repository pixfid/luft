# USB.IDS REGEX

### Version usb.ids
```Version: (\d{4}.\d{2}.\d{2})```

### Date
```Date:\s+(\d{4}-\d{2}-\d{2}) (\d{2}:\d{2}:\d{2})```

---
### List of known device classes, subclasses and protocols
#### Syntax:
#### C class class_name
####	subclass  subclass_name			<-- single tab
####		protocol  protocol_name		<-- two tabs

```^(C)\s+([[:xdigit:]]{2})\s+(.*)```

```^\t([[:xdigit:]]{2})\s+(.*)```

```^\t\t([[:xdigit:]]{2})\s+(.*)```

---
### List of Audio Class Terminal Types
#### Syntax:
#### AT terminal_type  terminal_type_name
```^(AT)\s+([[:xdigit:]]{4})\s+(.*)```

---
### List of HID Descriptor Types

### Syntax:
#### HID descriptor_type  descriptor_type_name
```^(HID)\s+([[:xdigit:]]{2})\s+(.*)```

---
### List of Physical Descriptor Bias Types

#### Syntax:
#### BIAS item_type  item_type_name
```^(BIAS)\s+([[:xdigit:]]{1})\s+(.*)```

___
### List of Physical Descriptor Item Types

#### Syntax:
#### PHY item_type  item_type_name
```^(PHY)\s+([[:xdigit:]]{2})\s+(.*)```

---
### List of HID Descriptor Item Types
#### Note: 2 bits LSB encode data length following
#### Syntax:
#### R item_type  item_type_name
```^(R)\s+([[:xdigit:]]{2})\s+(.*)```

---
### List of HID Usages

#### Syntax:
#### HUT hi  _usage_page  hid_usage_page_name
####	hid_usage  hid_usage_name
```^(HUT)\s+([[:xdigit:]]{2})\s+(.*)```

```^\t([[:xdigit:]]{3})\s+(.*)```

---
### List of Languages
#### Syntax:
#### L language_id  language_name
####	dialect_id  dialect_name
```^(L)\s+([[:xdigit:]]{4})\s+(.*)```

```^\t([[:xdigit:]]{2})\s+(.*)```

---
### HID Descriptor bCountryCode
#### HID Specification 1.11 (2001-06-27) page 23
#### Syntax:
#### HCC country_code keymap_type
```^(HCC)\s+(\d{2})\s+(.*)```

---
### List of Video Class Terminal Types
#### Syntax:
#### VT terminal_type  terminal_type_name
```^(VT)\s+(\d{4})\s+(.*)```