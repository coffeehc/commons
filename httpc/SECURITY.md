# HTTP defaults

Both BuildTransport and BuildDNSCacheTransport verify server certificates and
require TLS 1.2 or newer. Add an approved RootCAs pool for private trust; callers
remain responsible for deliberate custom transport settings.

ClientInitSetting retains an injected CookieJar. Its default null jar discards
cookies, and its default retry count is zero. Enable retries explicitly only when
requests are safe to repeat or the destination supplies deduplication.
A nil logger selects zap.NewNop. Existing public constructors remain available.

The security regression suite uses local TLS and cookie servers without real
credentials or external services.
