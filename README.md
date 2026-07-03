# epub2mobi

CLI in Go per convertire una cartella di ebook `.epub` in `.mobi` o `.azw3`.

La conversione non viene implementata a mano: i formati Kindle sono complessi e la scelta affidabile e' usare Calibre tramite il comando `ebook-convert`.

## Prerequisiti

- Go 1.23 o superiore
- Calibre installato e comando `ebook-convert` disponibile nel `PATH`

Su macOS:

```bash
brew install --cask calibre
```

Se `ebook-convert` non e' nel `PATH`, passa il percorso completo con `-converter`.

## Esecuzione

Da questa cartella:

```bash
go run ./cmd/epub2mobi -in ../Ebooks -out ../converted
```

Opzioni principali:

```text
-in         cartella sorgente con file .epub (obbligatoria)
-out        cartella di output; default: <input>/<format>
-format     formato output: mobi o azw3; default: mobi
-recursive  cerca .epub anche nelle sottocartelle
-overwrite  rigenera i .mobi gia' presenti
-jobs       numero conversioni parallele; default: 1
-mobi-type  tipo MOBI generato: old, new o both; default: both
-calibre-arg argomento extra per ebook-convert; ripetibile
-dry-run    mostra cosa verrebbe convertito senza creare file
-converter  comando o percorso di ebook-convert; default: ebook-convert
-timeout    timeout per singolo ebook; default: 30m
-verbose    stampa l'output di ebook-convert anche in caso di successo
```

Esempi:

```bash
go run ./cmd/epub2mobi -in ~/Books/epub -recursive
go run ./cmd/epub2mobi -in ./ebooks -out ./mobi -jobs 2 -overwrite
go run ./cmd/epub2mobi -in ./ebooks -out ./azw3 -format azw3 -overwrite
go run ./cmd/epub2mobi -in ./ebooks -dry-run
```

## Immagini negli EPUB

Per default il tool passa a Calibre:

```text
--mobi-file-type both
```

`both` genera un MOBI con parte legacy e parte KF8 moderna. Questo aumenta la dimensione del file, ma di solito conserva meglio immagini, layout e compatibilita' con reader diversi.

Se un reader vecchio non digerisce il file, prova:

```bash
go run ./cmd/epub2mobi -in ./ebooks -out ./mobi -mobi-type old -overwrite
```

Se invece usi un reader moderno e le immagini continuano a mancare, preferisci AZW3:

```bash
go run ./cmd/epub2mobi -in ./ebooks -out ./azw3 -format azw3 -overwrite -verbose
```

Quando usi `-format azw3`, il tool non passa opzioni specifiche del vecchio MOBI.

Per passare opzioni Calibre aggiuntive:

```bash
go run ./cmd/epub2mobi -in ./ebooks -calibre-arg "--disable-font-rescaling"
```

## Build

```bash
go build -o bin/epub2mobi ./cmd/epub2mobi
go build -o bin/epub2mobi-gui ./cmd/epub2mobi-gui
```

Poi:

```bash
./bin/epub2mobi -in ./ebooks -out ./converted
./bin/epub2mobi-gui
```

## Interfaccia grafica

La GUI usa lo stesso motore della CLI e permette di scegliere:

- cartella EPUB
- cartella output
- formato `mobi` o `azw3`
- numero di jobs
- overwrite, recursive, verbose e dry-run
- comando `ebook-convert`

Per avviarla da sorgente:

```bash
go run ./cmd/epub2mobi-gui
```

## Test

```bash
go test ./...
```
