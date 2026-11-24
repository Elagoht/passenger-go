# passenger-go: collage ile yeniden yazım — tasarım

Tarih: 2026-11-24 · Dal: `collage-rewrite`

## Amaç ve neden

passenger-go, tek kullanıcılı, kendi sunucunda çalışan bir parola yöneticisi. Mevcut
sürümün üç sorunu var:

1. **Kasa, ana parolayla korunmuyor.** Tüm kayıtlar `.env` içindeki tek bir
   `AES_GCM_SECRET` ile şifreli; ana parola da hash'lenmemiş, aynı anahtarla geri
   çözülebilir biçimde saklanıyor. `passenger.db` + `.env` ele geçiren herkes her şeyi
   düz metin okur.
2. **Oturum deneyimi bozuk.** JWT 5 dakikada bitiyor, yenileme yok; sayfa sessizce
   bozuluyor, "oturum bitti" davranışı yok.
3. Yan bulgular: boş `JWT_SECRET` kabul ediliyor, cookie'de `HttpOnly`/`Secure`/`SameSite`
   yok, CSRF koruması yok, `account-card.js` içinde escape edilmemiş `innerHTML` (XSS),
   giriş denemesi sınırı yok, kurulum yarışı (ilk gelen sahiplenir), CSV dışa aktarma
   escape etmiyor.

Kullanıcı projeyi kendi framework'ü **collage** (`github.com/Elagoht/collage`, v1.0.1)
ile, **tek uygulama** olarak baştan yazmak istiyor — ayrı backend yok, gesso yok, JSON
API yok. Eski veritabanı kayıpsız taşınabilmeli.

## Hedefler

- collage ile sunucu tarafında render edilen tek bir web uygulaması.
- Kasa anahtarı ana paroladan türetilir; disk üzerinde (DB veya `.env`) kasayı çözmeye
  yetecek hiçbir şey durmaz.
- Eski `passenger.db` + eski `.env` → yeni DB'ye tek komutla taşıma; ana parola aynı
  kalır, kurtarma anahtarı yenilenir (eskisi eski DB'de düz metin durduğu için).
- Oturum her istekte uzar; bitmeden önce kullanıcı uyarılır ve uzatabilir; bittiğinde
  açık bir "oturumun sona erdi" mesajıyla giriş sayfasına, giriş sonrası kaldığı sayfaya
  döner.
- Eski özelliklerin web karşılıkları korunur (aşağıda liste).
- Yukarıdaki yan bulguların hepsi kapanır.

## Hedef dışı

- JSON API, `/api-docs` sayfası, CLI istemcisi (kullanıcı: "eskiden cli vardı, şimdi gerek yok").
- Çok kullanıcı, paylaşım, tarayıcı eklentisi.
- Tarayıcıda (istemci tarafında) şifreleme.
- Dış servisten favicon çekme (`icon.horse`) — her hesabın alan adını üçüncü tarafa
  sızdırıyordu; yerine baş harf rozeti. *(Benim kararım.)*

## Anti-hedefler (çalışsa bile başarısızlık sayılır)

- Kasa anahtarının (DEK) veya herhangi bir kaydın düz metninin diske yazılması (log dahil).
- DB + sunucu yapılandırması ele geçirildiğinde ana parola olmadan kayıtların okunabilmesi.
- Taşımada tek bir kaydın sessizce kaybolması.
- Kullanıcının uyarı almadan, yazdığı formla birlikte oturumdan atılması.

## Yaklaşımlar

| | Yaklaşım | Artı | Eksi |
|---|---|---|---|
| **A (seçilen)** | Ana paroladan türetilen anahtarla sarılmış rastgele kasa anahtarı; oturum boyunca sunucu belleğinde | İlk açığı kapatır; parola değişimi ucuz; kurtarma anahtarı gerçekten çalışır; collage'ın SSR modeline oturur | Sunucu yeniden başlarsa tekrar giriş gerekir |
| B | Eski model: `.env` içindeki tek anahtar | En basit, restart'ta oturum düşmez | Büyük açık olduğu gibi kalır |
| C | Tarayıcıda şifreleme (zero-knowledge) | En güçlü model | collage'ı anlamsızlaştırır; arama/güç/içe aktarma tamamen JS |

Kullanıcı **A**'yı seçti.

## Kriptografi

Tüm ilkel işlemler `internal/vault` paketinde; başka hiçbir paket doğrudan `crypto/*`
çağırmaz.

- **DEK** (data encryption key): kurulumda üretilen 32 rastgele bayt. Asla diske düz
  yazılmaz.
- **Parola KEK'i**: `Argon2id(passphrase, salt_pass, t=3, m=64 MiB, p=4, len=32)`.
  Parametreler DB'de saklanır (ileride yükseltilebilsin).
- **Kurtarma KEK'i**: aynı Argon2id, `salt_rec` ile, kurtarma anahtarı dizesi üzerinden.
- **Sarma**: `wrapped_dek_pass = AES-256-GCM(KEK_pass, DEK, aad="passenger/v1/dek/pass")`,
  `wrapped_dek_rec` aynı şekilde `aad=".../dek/rec"`. Nonce rastgele, çıktının başında.
- **Giriş doğrulaması** = sarılmış DEK'in açılabilmesi. Ayrı parola hash'i yok.
- **Kayıtlar**: her hesap tek bir JSON blob (`platform, identifier, passphrase, url,
  notes`), `AES-256-GCM(DEK, blob, aad="passenger/v1/account/"+id)`; rastgele nonce.
  `aad` içindeki id, bir kaydın şifreli metninin başka satıra kopyalanmasını engeller.
- **Tekillik** (eski `UNIQUE(platform, identifier)`): `lookup = HMAC-SHA256(K_idx,
  lower(trim(platform)) + "\x00" + identifier)`, `K_idx = HKDF-SHA256(DEK, info="passenger/v1/index")`.
  `lookup` sütunu `UNIQUE`. Deterministik şifreleme artık yok.
- **Parola gücü** saklanmaz; gösterimde çözülmüş paroladan hesaplanır (eski
  `strength.go` mantığı taşınır).
- **Kurtarma anahtarı** (yeni kurulum): 32 rastgele bayt, base32 (padding yok), 4'lü
  gruplar halinde gösterilir; karşılaştırmada boşluk/tire/büyük-küçük harf yok sayılır.
  Her zaman yeni üretilir; eski biçim desteklenmez.
- **Kurtarma anahtarı yenileme**: DEK yeni bir kurtarma anahtarıyla yeniden sarılır
  (`salt_rec`, `wrapped_dek_rec` değişir); eski anahtar artık hiçbir şey açmaz.
- Bellekteki DEK oturum bitince/çıkışta sıfırlanır (best-effort `clear`).

## Veri modeli (SQLite, `modernc.org/sqlite`)

```sql
-- PRAGMA user_version = 1
CREATE TABLE vault (
  id               INTEGER PRIMARY KEY CHECK (id = 1),
  kdf_params       TEXT NOT NULL,   -- JSON: {"alg":"argon2id","t":3,"m":65536,"p":4}
  salt_pass        BLOB NOT NULL,
  wrapped_dek_pass BLOB NOT NULL,
  salt_rec         BLOB NOT NULL,
  wrapped_dek_rec  BLOB NOT NULL,
  created_at       INTEGER NOT NULL
);
CREATE TABLE accounts (
  id         TEXT PRIMARY KEY,      -- 16 rastgele bayt, hex
  blob       BLOB NOT NULL,
  lookup     BLOB NOT NULL UNIQUE,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
```

`vault` satırı yoksa uygulama "kurulmamış" durumdadır. Hesap listesi tüm satırları
çözer (tek kullanıcı, yüzlerce kayıt ölçeği — sorun değil).

## Oturum

collage oturum yönetimi vermiyor; kendi middleware'imiz (`internal/session`).

- Bellekte `map[token]*Session{DEK, createdAt, lastSeen}`; mutex korumalı. Token: 32
  rastgele bayt, base64url. Diske yazılmaz → restart herkesi çıkarır (kabul edilen bedel).
- Cookie `passenger_session`: `HttpOnly`, `SameSite=Strict`, `Path=/`, `Secure` istek
  TLS ise (`collage.IsHTTPS(r)`). Süresiz (session cookie); geçerlilik sunucuda.
- **Boşta kalma süresi** 15 dk (`SESSION_IDLE`, Go duration), **mutlak süre** 12 saat
  (`SESSION_MAX`). Korunan her istek `lastSeen`'i günceller (kayan oturum).
- Arka planda dakikada bir süresi dolanları silen temizleyici; `Shutdown`'da durur.
- Parola değişince / kurtarmada: diğer tüm oturumlar silinir, mevcut oturum yeni token
  alır.
- Çıkış: `POST /logout`, oturumu siler, cookie'yi temizler.

### "Oturum bitti" davranışı

- Korunan bir sayfaya oturumsuz/süresi dolmuş istek → `303 /login?expired=1&next=<yol>`
  (oturum hiç yoksa `expired` parametresi yok). Giriş başarılıysa `next`'e döner;
  `next` yalnızca `/` ile başlayan, `//` veya `/\` ile başlamayan yerel yol olabilir.
- Layout, oturumun bitiş zamanını bir `data-` özniteliğine yazar. Küçük bir statik JS
  (`static/session.js`, inline değil — CSP için):
  - Bitişe 60 sn kala önce `GET /session` ile gerçek bitişi sorar (başka sekme uzatmış
    olabilir); hâlâ yakınsa üstte bir şerit gösterir: "Oturumun 1 dakika içinde
    kapanacak — Uzat / Çıkış". "Uzat" → `POST /session/extend` (fetch, `X-CSRF-Token`).
  - Bitiş geldiğinde `/login?expired=1&next=<mevcut yol>` adresine gider.
  - Form içeren sayfalarda (detay/yeni hesap) uyarı şeridi aynı; ayrıca form alanları
    değiştiyse şerit "kaydedilmemiş değişiklik var" der.
- Oturum bittikten sonra gönderilen bir form (ör. dizüstü uykudayken JS zamanlayıcısı
  çalışmadı) **kaybolur**: sunucu kimliği doğrulanmamış form verisini (parola içerebilir)
  saklamaz. Uyarı şeridi bunun olağan durumda önüne geçer. *(Benim kararım.)*
- Giriş sayfası `expired=1` ile "Oturumun sona erdi, tekrar giriş yap" mesajı gösterir.
- `GET /session` → `{"expiresAt": <unix ms>}` veya 401; bu istek oturumu **uzatmaz**.

## Sayfalar ve eylemler

Arayüz dili İngilizce, eskisi gibi. *(Benim kararım; eski arayüz İngilizceydi.)*
Görsel tarz eski uygulamanınki (koyu, Catppuccin renkleri) korunur; CSS eskiden taşınır
ve sadeleştirilir.

| Yol | Ne | Koruma |
|---|---|---|
| `GET/POST /setup` | İlk kurulum: kurulum kodu + parola + tekrar | sadece kurulmamışken |
| `GET/POST /setup/confirm` | Kurtarma anahtarını bir kez göster, geri yazdırarak doğrula; doğrulanınca vault yazılır | kurulum akışında |
| `GET/POST /login` | Giriş | sadece kurulmuş ve oturumsuz |
| `GET/POST /recover` | Kurtarma anahtarı + yeni parola; başarıda kurtarma anahtarı da yenilenir ve bir kez gösterilir (kullanılmış anahtar ifşa olmuş sayılır) | kurulmuş |
| `GET /` | Hesap listesi + arama (`?q=`, sunucuda filtre; JS varsa anlık filtre) | oturum |
| `GET/POST /accounts/new` | Yeni hesap (generator butonu ile) | oturum |
| `GET/POST /accounts/{id}` | Detay/düzenle; parola maskeli, göster/kopyala | oturum |
| `GET/POST /accounts/{id}/delete` | GET onay sayfası, POST siler (JS `confirm` yok) | oturum |
| `POST /accounts/{id}/passphrase` | Listedeki "kopyala" için JSON `{passphrase}`; `X-CSRF-Token` ile | oturum |
| `GET/POST /generate` | Parola üretici ve "alternator" (eski `generator` mantığı) | oturum |
| `GET/POST /import` | Firefox / Chromium / passenger CSV içe aktarma; başarısızları listeler | oturum |
| `GET/POST /export` | Ana parolayı tekrar sorar, sonra CSV indirir | oturum |
| `GET/POST /settings/passphrase` | Mevcut + yeni parola | oturum |
| `GET/POST /settings/recovery` | Mevcut parola → yeni kurtarma anahtarı üretir, bir kez gösterir | oturum |
| `POST /logout`, `GET /session`, `POST /session/extend` | yukarıda | oturum |
| `/static/*` | `collage` mount, `os.OpenRoot` veya `embed` | açık |

Notlar:
- Başarısız form → aynı sayfa 422 ile, girilen değerler ve hata mesajı (collage
  `ActionResult{Page: rc.Page}` deseni). Başarılı → 303.
- Eski `api-docs`, JWT, `/api/*` yok.
- Dışa aktarma biçimi eskisiyle aynı başlık (`platform,identifier,passphrase,url,notes`),
  ama `encoding/csv` ile doğru escape edilir. İçe aktarıcı bu biçimi de tanır, yani dışa
  aktarılan dosya geri içe aktarılabilir (test: gidiş-dönüş). *(Benim kararım.)*
- Tüm sayfalar dinamik; collage sayfa önbelleği **kapalı** (`Cache.Enabled=false`),
  yanıtlar `Cache-Control: no-store`.

## Güvenlik ayrıntıları

- **CSRF**: collage'ın yerleşik koruması; her formda `{{csrfToken}}`. `CSRF_KEY`
  ortam değişkeni zorunlu (boşsa başlangıçta hata — collage'ın "üret ve uyar"
  davranışına güvenilmez).
- **XSS**: tüm çıktı `html/template` ile; JS tarafında `innerHTML` yerine `textContent`.
- **Başlıklar** (middleware): `Content-Security-Policy: default-src 'self'; script-src
  'self'; style-src 'self'; img-src 'self' data:; frame-ancestors 'none'; base-uri
  'none'; form-action 'self'`, `Referrer-Policy: no-referrer`, `X-Content-Type-Options`.
- **Kaba kuvvet** *(benim kararım)*: `/login`, `/recover`, `/setup`, `/setup/confirm`,
  `/export` ve `/settings/passphrase` denemeleri **istemci IP'si başına** sayılır
  (`collage.ClientIP`, `TRUSTED_PROXIES` ile): 5 başarısızlıktan sonra üstel bekleme
  (1 sn, 2 sn, 4 sn … en fazla 5 dk). Bekleme süresince o IP'nin denemeleri doğru olsa
  bile reddedilir (429 + "N saniye sonra tekrar dene" mesajı içeren sayfa); başka IP'ler
  etkilenmez, böylece tek bir saldırgan sahibini dışarıda bırakamaz. Ek olarak global bir
  tavan: tüm IP'lerden toplam saatte 100 başarısız denemeden sonra her denemeye 2 sn
  sabit gecikme (dağıtık denemeyi yavaşlatır, kilitlemez). Başarılı deneme o IP'nin
  sayacını sıfırlar. Bellekte.
- **Kurulum yarışı**: kurulmamışken uygulama başlangıçta loga 8 haneli tek kullanımlık
  bir **kurulum kodu** yazar; `/setup` bunu ister. Kodu sadece sunucu loguna erişen bilir.
- Hassas veriler (parola, DEK, kurtarma anahtarı, çözülmüş kayıt) asla loglanmaz.
- Kurtarma anahtarı gösterilen sayfa `no-store`.

## Taşıma (migration)

Uygulama ikilisinin bir alt komutu (collage `app.RegisterCommand`):

```
passenger-go migrate --from /opt/passenger-go/database/passenger.db \
                     --env  /opt/passenger-go/.env \
                     --to   ./data/passenger.db
```

1. `--env` dosyasından `AES_GCM_SECRET` okunur (32 bayt değilse hata).
2. Eski DB **salt okunur** açılır (`mode=ro`); eski dosyaya hiçbir şey yazılmaz.
3. Eski `user` satırı: `validated` değilse hata. `passphrase` çözülür → ana parola;
   `recovery` kullanılmaz.
4. Eski şifre çözme: hem "deterministik" hem rastgele alanlar aynı biçimde
   (`base64(nonce‖ciphertext)`, aynı anahtar) → tek bir `oldDecrypt` yeterli. `NULL`
   veya boş `url`/`notes` boş dize olur. `strength` atılır.
5. Yeni vault: yeni DEK, ana parola ile ve **yeni üretilen** kurtarma anahtarıyla sarılır.
   Yeni kurtarma anahtarı sonunda terminale bir kez basılır ("bunu güvenli bir yere
   kaydet; eski kurtarma anahtarın artık geçersiz").
6. Tüm hesaplar tek transaction içinde yeni şemaya yazılır. Herhangi bir kayıt
   çözülemezse **tüm taşıma iptal**, hedef dosya silinir, hangi eski `id`'nin
   bozuk olduğu yazılır.
7. `--to` zaten varsa reddeder (`--force` yok).
8. Doğrulama: hedefi yeniden açar, ana parolayla DEK'i açar, kayıt sayısını ve her
   kaydın eski karşılığıyla eşitliğini kontrol eder. Özet yazar: "N hesap taşındı.
   Ana parolan aynen geçerli; yeni kurtarma anahtarın: …"

Taşıma eski dosyaları silmez; sonunda açık bir uyarı yazar: eski `passenger.db` ve
eski `.env` (özellikle `AES_GCM_SECRET`) silinmeden kasa koruması tam değildir.
*(Benim kararım: yıkıcı adımı otomatikleştirmiyoruz.)*

Aynı (platform, identifier) çifti eskide de `UNIQUE` olduğu için çakışma beklenmez;
olursa (büyük/küçük harf farkı yüzünden yeni `lookup` çakışırsa) taşıma iptal edilir
ve çakışan eski `id`'ler listelenir.

## Yapılandırma

| Değişken | Varsayılan | Not |
|---|---|---|
| `PORT` | 8080 | |
| `DATABASE_PATH` | `./data/passenger.db` | dizin yoksa oluşturulur (0700), dosya 0600 |
| `CSRF_KEY` | — | zorunlu, en az 32 bayt |
| `SESSION_IDLE` | `15m` | |
| `SESSION_MAX` | `12h` | |
| `TRUSTED_PROXIES` | boş | collage `Server.TrustedProxies` |

`JWT_SECRET`, `AES_GCM_SECRET`, `SALT` artık yok.

## Proje yapısı ve dağıtım

- collage iskeleti düzeni (`collage new` ile aynı: `main.go`, `routes.go`, `pages/`,
  `fragments/`, `actions/`, `guards/`, `commands/`, `templates/`), alan mantığı
  `internal/vault`, `internal/session`, `internal/store`, `internal/importer`,
  `internal/generator`, `internal/strength`, `internal/legacy` (taşıma).
  Eski `backend/`, `frontend/`, `cli/` silinir.
- Şablonlar ve statik dosyalar mümkünse `embed` ile ikiliye gömülür, böylece
  `install.sh` dosya kopyalamaz. collage `TemplateConfig` bir `fs.FS` kabul etmiyorsa
  diskten okunur ve `install.sh` kopyalamaya devam eder — **kurucunun kontrol edeceği nokta**.
- `install.sh`, `passenger-go.service`, `.env.example`, `README.md`, release workflow
  yeni değişkenlere ve taşıma adımına göre güncellenir. `install.sh` `CSRF_KEY`'i kendi
  üretir (`openssl rand`), kullanıcıya sormaz. `.env`'i `>>` ile değil baştan yazar.
  `/opt/passenger-go/database/passenger.db` varsa (eski kurulum) servisi başlatmadan
  önce durur ve `migrate` komutunu ve eski dosyaların silinmesini anlatan talimatı
  yazar; taşımayı kendisi yapmaz. *(Benim kararım.)*
- Go 1.26.11+ (collage gereği).

## Testler

- `internal/vault`: sarma/açma, yanlış parola, kurtarma, parola değişimi, `aad`
  karıştırma reddi, lookup kararlılığı.
- `internal/legacy`: test içinde eski şifreleme biçimiyle üretilmiş sahte bir eski DB
  (eski `encrypt.go` mantığı test yardımcısı olarak kopyalanır) → taşıma → tüm
  kayıtların eşitliği; bozuk kayıtta tam iptal; doğrulanmamış kullanıcıda ret.
- `collagetest` ile uçtan uca: kurulum (kod dahil), giriş/çıkış, `next` yönlendirmesi
  ve açık yönlendirme reddi, süresi dolmuş oturum → `expired=1`, hesap CRUD, tekillik
  hatası, içe/dışa aktarma (virgül/tırnak içeren alanlar), parola değişimi diğer
  oturumları düşürür, CSRF'siz POST reddi, kaba kuvvet beklemesi.
- Zaman bağımlı testler için enjekte edilebilir saat.

## Kurucuya kalanlar

- collage API'sinin ayrıntıları (guard'ların sayfa/eylem bağlanışı, `RegisterCommand`
  imzası, `embed` desteği) — dokümana (`docs/*.md`, klonlanmış kaynak) bakarak.
- CSS'in sadeleştirilmesi ve sayfa düzenleri, eski görünümü koruyarak.
- Generator/alternator ve importer'ın birebir taşınması (davranış korunur, eski
  hatalar — ör. CSV escape — düzeltilir).
