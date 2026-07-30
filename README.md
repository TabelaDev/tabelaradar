<div align="center">

# ccdi

**TUI que fiscaliza a saúde git dos seus repositórios locais** — WIP, commits
não enviados, repos sem remote, projetos parados há tempo demais.

[![Go Version](https://img.shields.io/github/go-mod/go-version/ianptkcs/ccdi?style=flat-square&logo=go&logoColor=white&color=00ADD8)](go.mod)
[![License: PolyForm Noncommercial](https://img.shields.io/badge/license-PolyForm%20Noncommercial-blue?style=flat-square)](LICENSE)
[![Built with Bubble Tea](https://img.shields.io/badge/built%20with-Bubble%20Tea-ff69b4?style=flat-square)](https://github.com/charmbracelet/bubbletea)

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
go install github.com/ianptkcs/ccdi@latest
```

Ou compilando a partir do source:

```bash
git clone https://github.com/ianptkcs/ccdi.git
cd ccdi
go build -o ccdi .
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
ccdi         # abre a TUI
ccdi list    # dump em texto plano, sem TTY — útil pra scriptar
```

Dentro da TUI: `↑`/`↓` (ou `j`/`k`) navegam a lista de projetos,
`ctrl+h`/`ctrl+l` alternam entre o painel de projetos e o de descrição,
`j`/`k` (ou `↑`/`↓`) rolam o texto da descrição quando ela está focada,
`o`/`enter` abre o projeto selecionado no `$EDITOR` (padrão `nvim`), `r`
reescaneia, `q` sai.

## IPC

Pra scripts ou pra um LLM perguntar "o que falta fazer, onde eu parei em
cada projeto, o que dá pra começar a implementar" sem abrir a TUI, `ccdi`
expõe um subcomando `ipc` não-interativo, no mesmo espírito do `dcal ipc
<método> --json`/`djobs ipc <método> --json`:

```bash
ccdi ipc projects.list --json              # todo projeto trackeado, com status git + descrição + próximos passos
ccdi ipc projects.list dirty=true --json   # só quem tem mudanças não commitadas
ccdi ipc projects.list name=ndrc --json    # um projeto específico
ccdi ipc projects.next --json              # o projeto que a própria ccdi priorizaria (mid-flight > mais recente)
```

Cada projeto no JSON traz, além dos campos de status git (branch, sujo,
ahead/behind, último commit), `description` (extraída do README/PLANNING/
etc.), `memory_notes` (os hooks de uma linha do índice de memória) e
`next_steps` — o corpo **inteiro** (não só o hook truncado) de qualquer
memória daquele projeto marcada `type: next-steps` na sua própria
`~/.claude/projects/<slug>/memory/` — vazio se o projeto ainda não tiver
uma.

## Configuração

### Quais repos monitorar

`~/.config/ccdi/config` (padrão do `os.UserConfigDir()`; sobrescrível via
`CCDI_CONFIG`) lista, uma entrada por linha, o que entra no scan:

- um caminho para uma pasta-de-repos: cada subpasta com `.git` vira uma
  linha na tabela (é como `~/codigo/pessoal` funciona hoje);
- um caminho apontando direto pra um repositório git: ele entra sozinho,
  como uma linha só (útil pra monitorar um repo solto fora das pastas
  padrão, ex. `~/codigo/cogu`);
- `!<caminho>` exclui esse caminho específico — seja uma raiz inteira, seja
  um filho de uma raiz-de-repos citada acima.

Linhas em branco e começando com `#` são ignoradas. Sem esse arquivo, o
comportamento é o de sempre: varre só `CCDI_ROOT` (ou `~/codigo/pessoal`).

Exemplo:

```
~/codigo/pessoal
!~/codigo/pessoal/spotdash
~/codigo/cogu
```

### Outras variáveis

- `CCDI_ROOT` — diretório varrido quando não existe `~/.config/ccdi/config`
  (padrão `~/codigo/pessoal`).
- `CCDI_ACCENT` — accent Catppuccin Mocha manual, usado só quando o
  DankMaterialShell não está instalado/configurado (padrão `mauve`).
- `CCDI_DMS_SETTINGS` — caminho do `settings.json` do DMS, se não for
  o padrão.

## Licença

[PolyForm Noncommercial License 1.0.0](LICENSE) — uso livre pra fins não
comerciais (estudo, hobby, contribuição). Uso comercial requer permissão
do autor.
