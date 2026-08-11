<div align="center">

# TabelaRadar

**TUI que fiscaliza a saúde git dos seus repositórios locais** — WIP, commits
não enviados, repos sem remote, projetos parados há tempo demais.

[English](README.md) · **Português**

[![Go Version](https://img.shields.io/github/go-mod/go-version/TabelaDev/tabelaradar?style=flat-square&logo=go&logoColor=white&color=00ADD8)](go.mod)
[![License: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-blue?style=flat-square)](LICENSE)
[![Built with Bubble Tea](https://img.shields.io/badge/built%20with-Bubble%20Tea-ff69b4?style=flat-square)](https://github.com/charmbracelet/bubbletea)
[![Powered by tabelatuiui](https://img.shields.io/badge/theme-tabelatuiui-d6b4f7?style=flat-square)](https://github.com/TabelaDev/tabelatuiui)

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/ianptkcs)

</div>

---

## O que é

Um [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI que varre
os repositórios/pastas-de-repositórios listados nas settings (por padrão só
`~/codigo/pessoal`) e fiscaliza o estado de cada projeto: o que está em WIP,
o que tem commits não enviados, o que não tem remote configurado (logo, sem
backup fora da máquina) e o que já não é tocado há um tempo — a inspeção
disciplinar dos seus repositórios.

Não existe nenhum arquivo de metadado separado — todo o estado é inferido
do próprio git de cada repositório, mais o primeiro parágrafo de prosa do
seu `README.md`/`PLANNING.md`/`ESCOPO.md`/`STACK.md`/`TODO.md`/`CLAUDE.md`
(o primeiro que existir), mais os bullets do índice de memória do Claude
Code daquele projeto (`~/.claude/projects/<slug>/memory/MEMORY.md`), quando
uma sessão já rodou ali dentro.

O tema e o chrome compartilhado (header/footer/panels, padding ANSI-aware, os
helpers de `ipc ... --json`) vêm da
[`tabelatuiui`](https://github.com/TabelaDev/tabelatuiui), a lib de UI
compartilhada dos meus TUIs Bubble Tea.

## Índice

- [Instalação](#instalação)
- [Layout](#layout)
- [Uso](#uso)
- [IPC](#ipc)
- [Configuração](#configuração)
- [Licença](#licença)

## Instalação

Requer Go 1.26+.

```bash
go install github.com/ianptkcs/tabelaradar@latest
```

Ou compilando a partir do source:

```bash
git clone https://github.com/TabelaDev/tabelaradar.git
cd tabelaradar
go build -o tabelaradar .
```

## Layout

Três painéis:

- **Projetos** (esquerda, 1/5 da largura) — só os nomes, pra navegar rápido
  pela lista inteira. O glyph antes do nome resume o status: `○` sem git,
  `●` com mudanças não commitadas, `▲` com commits não enviados, `✕` com
  commits mas sem remote, `✓` limpo e sincronizado.
- **status** (topo direita, 1/5 da altura) — sujo (arquivos não
  commitados), push (commits não enviados) e atividade (há quanto tempo
  desde o último commit) do projeto selecionado.
- **descrição** (embaixo, 4/5 da altura) — o resto: path, último commit,
  avisos (sem remote, stash), descrição extraída do README/PLANNING/etc. e
  bullets da memória do Claude Code daquele projeto. Quando o conteúdo não
  cabe, o título mostra o intervalo visível (`descrição (1–13/27)`) e dá
  pra rolar.

## Uso

```
tabelaradar         # abre a TUI
tabelaradar list    # dump em texto plano, sem TTY — útil pra scriptar
```

Dentro da TUI: `↑`/`↓` (ou `j`/`k`) navegam a lista de projetos,
`ctrl+h`/`ctrl+l` alternam entre o painel de projetos e o de descrição,
`j`/`k` (ou `↑`/`↓`) rolam o texto da descrição quando ela está focada,
`o`/`enter` abre o projeto selecionado no `$EDITOR` (padrão `nvim`), `r`
reescaneia, `q` sai.

## IPC

Pra scripts ou pra um LLM perguntar "o que falta fazer, onde eu parei em
cada projeto, o que dá pra começar a implementar" sem abrir a TUI,
`tabelaradar` expõe um subcomando `ipc` não-interativo, no mesmo espírito do
`dcal ipc <método> --json`/`djobs ipc <método> --json`:

```bash
tabelaradar ipc projects.list --json                  # todo projeto trackeado, com status git + descrição + próximos passos
tabelaradar ipc projects.list dirty=true --json       # só quem tem mudanças não commitadas
tabelaradar ipc projects.list name=tabelacal --json   # um projeto específico
tabelaradar ipc projects.next --json                  # o projeto que o próprio tabelaradar priorizaria (mid-flight > mais recente)
```

Cada projeto no JSON traz, além dos campos de status git (branch, sujo,
ahead/behind, último commit), `description` (extraída do README/PLANNING/
etc.), `memory_notes` (os hooks de uma linha do índice de memória) e
`next_steps` — o corpo **inteiro** (não só o hook truncado) de qualquer
memória daquele projeto marcada `type: next-steps` na sua própria
`~/.claude/projects/<slug>/memory/` — vazio se o projeto ainda não tiver
uma.

## Configuração

Tudo fica em `~/.config/tabelaradar/config.toml` (sobrescrível via
`TABELARADAR_CONFIG`). O arquivo é opcional e parcial: só as chaves presentes
sobrescrevem, o resto segue no default. `f5` recarrega sem reiniciar.

```toml
roots = ["~/codigo/pessoal", "~/codigo/tabeladev"]
exclude = ["~/codigo/pessoal/spotdash"]

[scanner]
description_files = ["README.md", "PLANNING.md", "ESCOPO.md", "STACK.md", "TODO.md", "CLAUDE.md"]
claude_projects_dir = "~/.claude/projects"

[layout]
sidebar_width_share = 1  # razão de LARGURA sidebar:coluna-direita
right_width_share   = 4
stats_height_share  = 1  # razão de ALTURA stats:descrição
desc_height_share   = 4

[general]
editor = "nvim"  # vazio = usa $EDITOR, depois nvim
```

### Quais repos monitorar

`roots` aceita dois tipos de caminho:

- uma pasta-de-repos: cada subpasta com `.git` vira uma linha na tabela (é
  como `~/codigo/pessoal` funciona);
- um repositório git direto: entra sozinho, como uma linha só (útil pra um
  repo solto fora das pastas padrão).

`exclude` esconde um caminho específico — seja uma raiz inteira, seja um filho
de uma raiz citada em `roots`. A ordem não importa entre as duas listas.

Sem nenhum arquivo, varre só `TABELARADAR_ROOT` (ou `~/codigo/pessoal`).

### Migrando do formato antigo

Antes da 0.3.0 a config era `~/.config/tabelaradar/config`, uma entrada por
linha com `!` prefixando exclusões. **Esse arquivo continua sendo lido** quando
não existe `config.toml`, com um aviso na barra de status. A tradução é direta:

```
~/codigo/pessoal              →  roots   = ["~/codigo/pessoal"]
!~/codigo/pessoal/spotdash    →  exclude = ["~/codigo/pessoal/spotdash"]
```

Criado o `config.toml`, ele passa a valer sozinho — os dois formatos não se
misturam, e o arquivo antigo pode ser apagado.

### Outras variáveis

- `TABELARADAR_CONFIG` — caminho do `config.toml`, se não for o padrão.
- `TABELARADAR_ROOT` — diretório varrido quando não existe config nenhuma
  (padrão `~/codigo/pessoal`).
- `TABELARADAR_ACCENT` — accent Catppuccin Mocha manual, usado só quando o
  DankMaterialShell não está instalado/configurado (padrão `mauve`).
- `TABELARADAR_DMS_SETTINGS` — caminho do `settings.json` do DMS, se não for
  o padrão.

## Desenvolvimento

```bash
go test ./...
```

## Changelog

Veja [CHANGELOG.md](CHANGELOG.md) para o histórico de versões.

## Apoie o projeto

- **Global**: [ko-fi.com/ianptkcs](https://ko-fi.com/ianptkcs)
- **Brasil (Pix)**: escaneie o QR abaixo ou copie o código

  <img src="pix-qr.png" alt="Pix QR" width="200" />

  <details><summary>Código Pix (copiar)</summary>

  ```
  00020126580014BR.GOV.BCB.PIX01365ad933b0-dcdc-4525-a736-0759902aeec65204000053039865802BR5925Ian Patrick da Costa Soar6009SAO PAULO62140510tQA85x6Dov63041FB6
  ```

  </details>

## Licença

[GNU AGPL-3.0](LICENSE) — livre e open source. Se você rodar uma versão
modificada deste projeto, inclusive como serviço de rede, também precisa
disponibilizar o código-fonte modificado sob a mesma licença.
