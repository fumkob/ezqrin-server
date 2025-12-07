# Internationalization (i18n) Support

Complete guide to internationalization support in the ezQRin API.

## Overview

The ezQRin API supports content delivery in multiple languages and locales. Internationalization
enables:

- Multi-language error messages
- Localized email templates
- Regional date/time formatting
- Locale-specific communication

**Current Status:** i18n framework designed; implementation planned for Phase 2

---

## Supported Languages

### Tier 1 (Full Support)

- **English (en-US)** - Default language
- **Japanese (ja-JP)** - Full translation + cultural adaptations
- **Spanish (es-ES)** - Full translation

### Tier 2 (Planned)

- **French (fr-FR)** - Q2 2026
- **German (de-DE)** - Q2 2026
- **Simplified Chinese (zh-CN)** - Q2 2026
- **Korean (ko-KR)** - Q3 2026

### Tier 3 (On Demand)

Additional languages available through enterprise support.

---

## Language Selection

### 1. HTTP Header (Recommended)

Use `Accept-Language` header to specify preferred language:

```http
GET /api/v1/events/123/participants
Accept-Language: ja-JP, ja;q=0.9, en-US;q=0.8
```

**Language Precedence:**

- First language in list is primary preference
- `q` parameter indicates preference weight (0-1)
- Falls back to English if language not supported

### 2. Query Parameter (Alternative)

```
GET /api/v1/events/123/participants?lang=ja-JP
```

### 3. User Preference (Most Specific)

For authenticated users, language preference stored in user profile:

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "preferred_language": "ja-JP"
}
```

**Preference Priority:**

1. Explicit query parameter (`?lang=ja-JP`)
2. User profile preference
3. Accept-Language header
4. Event default language
5. English (en-US)

---

## Response Language

All responses include language information:

```json
{
  "success": true,
  "data": { ... },
  "language": "ja-JP",
  "locale": {
    "language_code": "ja",
    "country_code": "JP",
    "timezone": "Asia/Tokyo"
  }
}
```

---

## Localized Content

### Error Messages

All error messages translated:

**English (en-US):**

```json
{
  "error": {
    "code": "PARTICIPANT_NOT_FOUND",
    "message": "Participant not found"
  }
}
```

**Japanese (ja-JP):**

```json
{
  "error": {
    "code": "PARTICIPANT_NOT_FOUND",
    "message": "参加者が見つかりません"
  }
}
```

### Date & Time Formatting

Dates and times formatted according to locale:

**English (en-US):**

```json
{
  "event": {
    "start_date": "12/15/2025",
    "start_time": "09:00 AM"
  }
}
```

**Japanese (ja-JP):**

```json
{
  "event": {
    "start_date": "2025年12月15日",
    "start_time": "09:00"
  }
}
```

**ISO 8601 (Always Available):**

```json
{
  "event": {
    "start_date_iso": "2025-12-15T09:00:00Z"
  }
}
```

### Email Templates

Email templates translated and culturally adapted:

**English (en-US):**

```
Subject: Your QR Code for Tech Conference 2025
Hello John Doe,

Your event is coming up! Check in with the QR code below...
```

**Japanese (ja-JP):**

```
Subject: Tech Conference 2025 のQRコード
John Doeさんへ、

イベントが近づいています！以下のQRコードでチェックインしてください...
```

### Event Content

Event descriptions, agendas, and location information translated per-event:

**Create Multilingual Event:**

```
POST /api/v1/events
Content-Type: application/json
Accept-Language: en-US

{
  "name_en": "Tech Conference 2025",
  "name_ja": "テックカンファレンス2025",
  "description_en": "Annual technology conference...",
  "description_ja": "毎年開催されるテクノロジーカンファレンス...",
  "location_en": "San Francisco Convention Center",
  "location_ja": "サンフランシスコ・コンベンションセンター",
  "timezone": "America/Los_Angeles",
  "primary_language": "en-US",
  "supported_languages": ["en-US", "ja-JP"]
}
```

**Get Multilingual Event:**

```
GET /api/v1/events/123?lang=ja-JP
```

**Response:**

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "テックカンファレンス2025",
    "description": "毎年開催されるテクノロジーカンファレンス...",
    "location": "サンフランシスコ・コンベンションセンター",
    "language": "ja-JP"
  }
}
```

---

## Participant Portal Localization

Participant portal automatically adapts to user's language preference:

**Magic Link with Language:**

```
https://portal.ezqrin.com/login?token=magic_...&lang=ja-JP
```

**Participant Session Language:**

```
POST /api/v1/portal/auth/validate
Content-Type: application/json

{
  "token": "magic_...",
  "preferred_language": "ja-JP"
}
```

---

## Email Template Translation

### Built-in Template Translations

Default email templates available in all supported languages:

**Query:**

```
GET /api/v1/events/123/email-templates?language=ja-JP
```

**Response:**

```json
{
  "success": true,
  "data": [
    {
      "id": "tpl_default_jp",
      "name": "デフォルト",
      "language": "ja-JP",
      "subject": "Your QR Code for {event_name}",
      "is_system": true
    }
  ]
}
```

### Custom Template Translation

Create custom templates with multiple language versions:

```
POST /api/v1/events/123/email-templates
Content-Type: application/json

{
  "name_en": "Welcome Email",
  "name_ja": "ウェルカムメール",
  "translations": {
    "en-US": {
      "subject": "Welcome to {event_name}",
      "body": "<html>Welcome {participant_name}...</html>"
    },
    "ja-JP": {
      "subject": "{event_name}へようこそ",
      "body": "<html>{participant_name}さん、ようこそ...</html>"
    }
  }
}
```

### Automatic Language Selection for Email

Send emails in participant's preferred language:

```
POST /api/v1/events/123/qrcodes/send
Content-Type: application/json

{
  "participant_ids": ["770e8400..."],
  "email_template": "welcome",
  "use_participant_language": true
}
```

Sends email in each participant's preferred language (from profile).

---

## Date/Time Localization

### Timezone Support

Events support explicit timezone specification:

```json
{
  "event": {
    "start_date": "2025-12-15T09:00:00Z",
    "timezone": "America/Los_Angeles",
    "timezone_offset": "-08:00"
  }
}
```

### Locale-Aware Formatting

Dates/times formatted according to locale automatically:

**Query with Locale:**

```
GET /api/v1/events/123?lang=ja-JP&format=localized
```

**Response:**

```json
{
  "data": {
    "start_date": "2025年12月15日",
    "start_time": "午前9:00",
    "start_datetime": "2025年12月15日(日) 午前9:00"
  }
}
```

### ISO 8601 Always Available

Raw ISO format always available for programmatic use:

```json
{
  "data": {
    "start_date_iso": "2025-12-15T09:00:00-08:00",
    "start_date_utc": "2025-12-15T17:00:00Z"
  }
}
```

---

## Number Formatting

Numbers formatted per locale:

**English (en-US):**

```json
{
  "participant_count": "1,250",
  "fee": "$99.99"
}
```

**Japanese (ja-JP):**

```json
{
  "participant_count": "1,250",
  "fee": "¥9,999"
}
```

---

## Right-to-Left (RTL) Language Support

Future support for RTL languages (Arabic, Hebrew):

```json
{
  "locale": {
    "language": "ar-SA",
    "text_direction": "rtl"
  }
}
```

---

## Cultural Adaptations

### Language-Specific Customizations

Beyond translation, cultural adaptations:

**Japanese Events:**

- Honorific language for formal communication
- Respect for Japanese business customs
- Holiday and date considerations
- Proper name handling (family name first)

**Spanish Events:**

- Formal vs. informal address modes
- Regional variations (Spain vs. Latin America)
- Currency and measurement units

---

## Organization i18n Settings

Organizations can configure language preferences:

**Update Organization Settings:**

```
PUT /api/v1/organizations/org123
Content-Type: application/json

{
  "default_language": "ja-JP",
  "supported_languages": ["ja-JP", "en-US"],
  "email_language_mode": "participant_preference",
  "date_format": "localized"
}
```

---

## API Endpoint i18n Examples

### Get Event (Multilingual)

```
GET /api/v1/events/123?lang=ja-JP
Accept-Language: ja-JP
```

### List Participants (Localized Error)

```
GET /api/v1/events/999/participants
Accept-Language: ja-JP
```

**Response (when event not found):**

```json
{
  "success": false,
  "error": {
    "code": "EVENT_NOT_FOUND",
    "message": "イベントが見つかりません"
  },
  "language": "ja-JP"
}
```

### Send Emails (Multilingual)

```
POST /api/v1/events/123/qrcodes/send
Accept-Language: ja-JP
Content-Type: application/json

{
  "participant_ids": ["..."],
  "email_template": "default",
  "language": "ja-JP"
}
```

---

## Implementation Notes

### Language Code Format

Follow IANA Language Subtag Registry:

- Main language: ISO 639-1 code (e.g., "en", "ja")
- Optional region: ISO 3166-1 alpha-2 code (e.g., "US", "JP")
- Format: `language-REGION` (e.g., "ja-JP", "en-US")

### Fallback Strategy

```
1. Requested language (from header/query/profile)
2. Event's primary language
3. Organization's default language
4. English (en-US)
```

### Translation Management

**Planned tools for Phase 2:**

- Crowdsourced translation
- Professional translation provider integration
- Translation status dashboard
- Missing translation detection

### Performance Optimization

- Cache translations per language
- Use CDN for localized static content
- Lazy-load translations for rarely-used languages
- Database indexes on language fields

---

## Best Practices

### For API Consumers

1. **Specify Language Early:**
   - Set `Accept-Language` header on first request
   - Store user's language preference
   - Pass language in magic links

2. **Handle Multiple Languages:**
   - Support fallback to English
   - Display original language if translation unavailable
   - Use ISO 8601 for dates (always)

3. **Test Localization:**
   - Test with multiple locales in sandbox mode
   - Verify date/number formatting
   - Check translated error messages

### For Event Organizers

1. **Create Multilingual Events:**
   - Provide translations for event details
   - Create language-specific email templates
   - Consider timezone for global events

2. **Participant Communication:**
   - Use participant's preferred language
   - Provide contact info in multiple languages
   - Include language option in confirmation

### For Integrators

1. **Language Detection:**
   - Use browser `Accept-Language` header
   - Cache user's language preference
   - Allow manual language selection

2. **Content Adaptation:**
   - Format dates per locale
   - Translate error messages for display
   - Adapt UI to text direction (future RTL support)

---

## Future Enhancements

**Phase 2 (2026):**

- Additional Tier 2 languages (French, German, Chinese, Korean)
- Crowdsourced translation community
- Professional translation provider integration
- Translation quality metrics

**Phase 3 (2027):**

- RTL language support (Arabic, Hebrew)
- Colloquial/dialect support
- Advanced date/time customization
- Automated translation with human review

---

## Language Support Status

| Language             | Code  | Status             | Scope                               |
| -------------------- | ----- | ------------------ | ----------------------------------- |
| English (US)         | en-US | ✅ Full            | API, errors, emails, templates      |
| Japanese             | ja-JP | ✅ Full            | API, errors, emails, templates      |
| Spanish              | es-ES | ✅ Full            | API, errors, emails, templates      |
| French               | fr-FR | 🔲 Planned Q2 2026 | API, errors, emails, templates      |
| German               | de-DE | 🔲 Planned Q2 2026 | API, errors, emails, templates      |
| Chinese (Simplified) | zh-CN | 🔲 Planned Q2 2026 | API, errors, emails, templates      |
| Korean               | ko-KR | 🔲 Planned Q3 2026 | API, errors, emails, templates      |
| Arabic               | ar-SA | 🔲 Planned Phase 3 | API, errors, emails, templates, RTL |
| Hebrew               | he-IL | 🔲 Planned Phase 3 | API, errors, emails, templates, RTL |

---

## Related Documentation

- [API Documentation Index](./README.md)
- [Email Templates](./qrcode.md#email-templates)
- [Error Codes](./error_codes.md)
- [QR Code API - Participant Access](./qrcode.md#participant-portal-access)
- [Testing Guide](./testing.md)
