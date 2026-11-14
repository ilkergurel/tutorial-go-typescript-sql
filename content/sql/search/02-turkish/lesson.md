---
id: sql.search.turkish
title: "The turkish, english and simple FTS configurations; Turkish text gotchas"
track: sql
module: search
order: 2
level: advanced
minutes: 14
since: pg12
prerequisites: [sql.search.fts]
compare_with: [sql.search.trigram, sql.aggregation.group-by, sql.tables-types.text-numbers, go.basics.strings-runes]
tags: [full-text-search, turkish, stemming, snowball, ts_debug, collation, tr-x-icu]
summary: >
  A query must use its document's configuration: turkish stems Turkish suffixes,
  english and simple barely touch Turkish words, and a mismatch finds nothing
  without an error. to_tsvector lowercases with the database's rules, so İ/ı need
  lower(... COLLATE "tr-x-icu") on both sides.
examples:
  - id: configs
    file: examples/01_configs.sql
    title: "One sentence, three configurations, and a query in the wrong one"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-3"
        note: >
          turkish strips plural and case endings (raporlar, testleri) and the
          past tense (hazırlandı, bitti), and drops ve and the da split off
          İstanbul'da as stop words. english and simple produce the same vector
          for this sentence: neither knows Turkish stop words, and the English
          stemmer finds nothing English to strip here.
      - lines: "8"
        note: >
          The column holds turkish stems, with the configuration named, as an
          index or generated column requires.
      - lines: "15-16"
        note: >
          initdb sets default_text_search_config from the server's locale:
          english on this English-language Windows machine, possibly turkish on
          a server installed with a Turkish locale, and the built-in simple when
          no configuration matches. SET LOCAL pins it for this transaction, so
          the example behaves the same on any server.
      - lines: "22-23"
        note: >
          A query that names no configuration is parsed with the default,
          english. rapor happens to come out unchanged and works; raporlar
          stays raporlar and şifre becomes şifr, and neither is in the column.
          No error, no warning, just missing rows, and a test that only
          searches for base forms passes.
      - lines: "24-25"
        note: >
          simple only lowercases, so it finds a row only when the user happens
          to type the stem itself: rapor and şifre.
  - id: dotted-i
    file: examples/02_dotted_i.sql
    title: "İ/ı: lowercase the Turkish way, on both sides"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-6"
        note: >
          The turkish configuration lowercases each word with the database's
          rules before stemming. Here (builtin C.UTF-8) both I and İ become i:
          IĞDIR turns into iğdir, which the stemmer cuts to ik; ığdır becomes
          ık; mixed Iğdır becomes iğdır and is left alone. One city, three
          lexemes. lower(... COLLATE "tr-x-icu") maps I to ı and İ to i first,
          and all three spellings meet at ık.
      - lines: "5"
        note: >
          The flip side is IZMIR, typed on an English keyboard: Turkish rules
          make it ızmır, which no longer matches İzmir. Each lowering is right
          for one way of typing and wrong for the other.
      - lines: "9"
        note: >
          to_tsvector ignores the collation of its argument: ISPARTA still
          becomes ispar. A column declared COLLATE "tr-x-icu" doesn't make
          full-text search Turkish; lower() is what reads the collation.
      - lines: "15-16"
        note: >
          lower() is immutable, with or without the COLLATE, so the Turkish
          lowering can live in a generated column and under an index.
      - lines: "23-29"
        note: >
          The query must be lowered the same way as the column. raw_query works
          only when the user typed lowercase Turkish; lowered, every Turkish
          spelling finds its rows. igdir, typed without Turkish letters, finds
          nothing either way: no lowering turns g into ğ. That case belongs to
          unaccent and pg_trgm, in the next lesson.
  - id: stemmer-limits
    file: examples/03_stemmer_limits.sql
    title: "Where the turkish stemmer goes wrong, and prefixes as a second net"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-5"
        note: >
          rapor is the stemmer at its best: three forms, one lexeme. hata and
          gereksinim show its failure mode. The bare word ends in letters that
          look like a suffix (-ta as in kitapta, -im as in kitabım), so hata
          is cut to ha and gereksinim to gereks, while hatalar and gereksinimler
          are cut back to the real root. Singular and plural end up as
          different lexemes.
      - lines: "8-9"
        note: >
          ts_debug shows each token, its type, the dictionary that took it and
          the result. The apostrophe is a separator, so TraceLite'ın yields
          traceli (the final -te is read as a suffix too) and ın, and
          İstanbul'da yields istanbul and the stop word da. Proper-noun suffixes
          become loose lexemes like ın.
      - lines: "18-20"
        note: >
          A stem search for hata finds only the singular. A prefix on unstemmed
          words finds every form, because Turkish adds suffixes at the end, and
          it also finds Hatay, a city. Prefixes trade misses for false hits;
          rank stem matches above prefix-only ones.
use_cases:
  - "TraceLite installations with Turkish requirement texts: turkish configuration over the description, with Turkish lowering"
  - "Names, codes and tags (people, cities, product codes): the simple configuration, no stemming"
  - "Search-as-you-type over Turkish titles: a simple vector for prefixes next to the stemmed one"
strengths:
  - "A built-in Turkish stemmer and stop word list: raporlar, raporları and raporların all find rapor with no extra setup, unlike LIKE"
  - "ts_debug shows exactly which token went through which dictionary, so a missed match can be explained rather than guessed at"
  - "Configurations are SQL objects: CREATE TEXT SEARCH CONFIGURATION ... (COPY = turkish) and ALTER MAPPING change behaviour per database without code"
weaknesses:
  - "The stemmer is rule-based: some common words split into several lexemes (hata/ha, gereksinim/gereks, kullanıcı/kullanıç)"
  - "No Turkish-aware lowercasing inside the configuration: it uses the database's case rules, which map I to i under C.UTF-8"
  - "Unlike Lucene's Turkish analysis chain, there is no apostrophe handling: suffixes after an apostrophe become separate lexemes"
  - "A wrong or missing configuration in a query raises no error; the only symptom is missing rows"
pitfalls:
  - title: "Querying in a different configuration than the document"
    detail: "The column was built with turkish; a query without a configuration uses default_text_search_config, which depends on how the server was installed. raporlar then finds nothing, silently."
    bad: "WHERE search @@ plainto_tsquery($1)"
    good: "WHERE search @@ plainto_tsquery('turkish', $1)"
  - title: "Expecting COLLATE to make full-text search Turkish"
    detail: "to_tsvector ignores the collation of its input and lowercases with the database's rules. Only lower() reads the collation, so lowercase explicitly before handing the text over."
    bad: "to_tsvector('turkish', title COLLATE \"tr-x-icu\")"
    good: "to_tsvector('turkish', lower(title COLLATE \"tr-x-icu\"))"
  - title: "Lowering only the document"
    detail: "With Turkish lowering in the column, the column holds ık for IĞDIR, but plainto_tsquery('turkish', 'IĞDIR') produces ik. Put the lowering in one SQL function and call it for the column and for every query."
    bad: "WHERE search @@ plainto_tsquery('turkish', $1)"
    good: "WHERE search @@ plainto_tsquery('turkish', lower($1 COLLATE \"tr-x-icu\"))"
  - title: "Putting unaccent in front of the turkish stemmer"
    detail: "The unaccent docs show ALTER MAPPING ... WITH unaccent, french_stem. The same recipe weakens the Turkish stemmer, whose rules read ı, ğ, ö and ü: tested here, gözlüklerini becomes gozlukler instead of gözlük, ağacı stays agaci instead of ağaç, and ığdır becomes igi. Keep the stemmed vector Turkish and add a folded one next to it if users type without Turkish letters."
  - title: "Trusting the stemmer with your domain vocabulary"
    detail: "hata (error) and gereksinim (requirement) are the words a TraceLite user searches for most, and both split into several lexemes. Run your vocabulary through to_tsvector('turkish', word) once and add a prefix match or a synonym dictionary where it fails."
quiz:
  - q: "The column holds to_tsvector('turkish', title). The app runs WHERE search @@ plainto_tsquery($1) on a server whose default configuration is english. What does a search for raporlar return?"
    options: ["An error: the configurations differ", "The matching rows: Postgres converts the query to turkish", "Nothing, silently: the query keeps raporlar, the column holds rapor"]
    answer: 2
    why: "Configurations are applied, not checked. The english stemmer leaves raporlar alone, and that lexeme isn't in the column."
  - q: "In this database (builtin C.UTF-8), what does to_tsvector('turkish', 'IŞIK') contain?"
    options: ["'ışık'", "'işik'", "'isik'"]
    answer: 1
    why: "The configuration lowercases with the database's rules, which map I to i; ş lowercases normally. lower('IŞIK' COLLATE \"tr-x-icu\") would give ışık."
  - q: "Why does a turkish-configuration search for hata miss a requirement titled 'Raporlama modülündeki hatalar'?"
    options: ["hatalar is a stop word", "The stemmer cuts hata to ha but hatalar to hata, so query and document get different lexemes", "Turkish words need a GiST index"]
    answer: 1
    why: "The stemmer works by suffix rules; when a bare word ends like a suffix, it is cut too far."
  - q: "A user types IZMIR on an English keyboard. Which lowering, applied to both sides, lets it match İzmir?"
    options: ["lower(q COLLATE \"tr-x-icu\")", "The database default, C.UTF-8", "Neither"]
    answer: 1
    why: "C.UTF-8 maps I and İ to i, so both become izmir. Turkish rules turn IZMIR into ızmır. For users who type both ways, the next lesson folds ı and i together."
further_reading:
  - title: "PostgreSQL docs: Text Search Configuration"
    url: "https://www.postgresql.org/docs/current/textsearch-configuration.html"
  - title: "PostgreSQL docs: Dictionaries (simple, Snowball, synonym, Ispell)"
    url: "https://www.postgresql.org/docs/current/textsearch-dictionaries.html"
  - title: "PostgreSQL docs: Testing and Debugging Text Search (ts_debug)"
    url: "https://www.postgresql.org/docs/current/textsearch-debugging.html"
  - title: "PostgreSQL docs: default_text_search_config"
    url: "https://www.postgresql.org/docs/current/runtime-config-client.html#GUC-DEFAULT-TEXT-SEARCH-CONFIG"
  - title: "PostgreSQL docs: Collation Support (ICU collations)"
    url: "https://www.postgresql.org/docs/current/collation.html"
---

## Concept

A configuration is the language of a search, and it has to be the same on both sides of
`@@`. Three matter for Turkish text:

- **`turkish`**: Snowball's Turkish stemmer and a Turkish stop word list. It removes
  plural, case, possessive and tense suffixes (*raporları* → `rapor`, *bitti* → `bit`)
  and drops words like *ve*, *da* and *için*.
- **`english`**: the English stemmer and stop words. On Turkish text it mostly does
  nothing, and now and then applies an English rule (*şifre* → `şifr`).
- **`simple`**: lowercase, nothing else. No stop words, no stemming: right for names,
  codes and tags, and as the prefix companion of a stemmed vector.

Three Turkish traps follow, and none of them raises an error:

1. **Configuration mismatch.** A document built with `turkish` and a query parsed with
   another configuration match only where both happen to produce the same lexeme.
2. **Lowercasing follows the database, not Turkish.** Each word is lowercased with the
   database's case rules before the stemmer sees it. StackLab's databases use the builtin
   `C.UTF-8` locale, which maps both `I` and `İ` to `i`, so *IĞDIR*, *Iğdır* and *ığdır*
   become three different lexemes. `lower(text COLLATE "tr-x-icu")`, applied to the
   column and to the query, fixes this for text typed with Turkish letters.
3. **The stemmer is suffix rules, not a dictionary.** It handles regular nouns well and
   cuts some words too far: *hata* becomes `ha` while *hatalar* becomes `hata`.

## Coming from C/C++/Java/Python/JS

- **Java:** `"TITLE".toLowerCase()` uses the JVM's default locale, and on a Turkish
  machine returns `tıtle`: the classic Turkish locale bug. PostgreSQL errs the other way.
  It never applies Turkish rules unless the expression names a Turkish collation.
  `toLowerCase(Locale.forLanguageTag("tr"))` corresponds to `lower(x COLLATE "tr-x-icu")`.
- **Go:** `strings.ToLower` applies Unicode's default mapping, like `C.UTF-8`;
  `strings.ToLowerSpecial(unicode.TurkishCase, s)` applies Turkish rules.
- **Python/JS:** `str.lower()` and `toLowerCase()` use Unicode's full case mapping, so
  `'İ'` becomes `i` plus a combining dot (two code points), while `C.UTF-8`'s simple
  mapping gives a plain `i`. Text lowered in the app and text lowered in Postgres can
  differ byte for byte: lowercase in one place. `toLocaleLowerCase('tr')` applies
  Turkish rules.
- **Lucene / Elasticsearch:** the Turkish analysis chain has a Turkish lowercase filter
  and an apostrophe filter that drops the suffix after an apostrophe. PostgreSQL's
  `turkish` configuration has neither: you add the lowercasing yourself, and
  *TraceLite'ın* leaves a lexeme `ın` behind.

## Design notes

**Choose per field.** Turkish prose gets `turkish`; names, codes and tags get `simple`.
For rows in different languages, store the configuration in a column:
`to_tsvector(lang, body)` with `lang regconfig` is still immutable and indexable. A query
then ORs the languages it might be in: `plainto_tsquery('turkish', $1) ||
plainto_tsquery('english', $1)`.

**Normalize in one place.** The Turkish lowering has to happen identically for the column
and for every query. Put it in an SQL function and call it on both sides, rather than
lowering in Go or TypeScript on one side and in SQL on the other.

**Stems and words, together.** Stems catch *kitabı* for *kitap* (consonant change, which
a prefix misses); prefixes catch *hatalar* for *hata* (which the stemmer misses). Keep
both, and rank by the stem match:

```sql
search_stems tsvector GENERATED ALWAYS AS
  (to_tsvector('turkish', lower(title COLLATE "tr-x-icu"))) STORED,
search_words tsvector GENERATED ALWAYS AS
  (to_tsvector('simple', lower(title COLLATE "tr-x-icu"))) STORED
-- GIN on both; $1 is the lowered input, $2 a prefix query built from its last word
WHERE search_stems @@ plainto_tsquery('turkish', $1)
   OR search_words @@ to_tsquery('simple', $2)
ORDER BY ts_rank(search_stems, plainto_tsquery('turkish', $1)) DESC, id
```

**Know your users' keyboards.** Turkish lowering is right when people type Turkish
letters. If many type `IZMIR` or `igdir`, no lowering helps; fold both sides to ASCII
for matching (the next lesson's `unaccent` key) and accept that *kır* and *kir* become
the same word.

**Custom dictionaries are possible, at a price.** A synonym dictionary can map your
domain's problem words, and Ispell/Hunspell dictionaries for Turkish exist, but their
files must live in the server's `tsearch_data` directory: an operations task, not
something an application migration can ship.
