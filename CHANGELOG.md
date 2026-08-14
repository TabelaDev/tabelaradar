# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.3.0] - 2026-08-14

### Adicionado

- **`digest`** — subcomando que vira a atividade dos projetos em updates no
  kanban (`tabelaradar digest`). Coleta commits/estado do git, memória do
  Claude e (opcional, off por default) sessões do opencode; lê o board via
  `tabelakanban ipc boards.list`; pede a um LLM um plano estruturado
  (`moves`/`updates`/`creates`) e aplica via `cards.move`/`cards.update`/
  `cards.create` — sem nada embutido no kanban, o mapeamento board→projetos é
  config do próprio radar. Tudo decisivo é configurável na seção `[digest]`
  (on/off, dry-run, 5 providers de LLM — opencode/claude CLIs e
  deepseek/openai/anthropic via API —, fontes de atividade, estado, schedule).
  `digest --dry-run` só imprime o plano; `digest --install-timer` cria o
  systemd user timer (`Persistent=true`). Requer `tabelakanban` ≥ v0.3.0 (o
  método `ipc cards.update`).
- Config em TOML (`~/.config/tabelaradar/config.toml`), substituindo o formato
  de uma-linha-por-caminho. Além de `roots`/`exclude`, agora são configuráveis
  os arquivos de descrição, o dir de memória do Claude, as proporções de
  layout e o editor.
- Tecla `f5`: recarrega config.toml e keybindings sem reiniciar.
- Primeiros testes do repo, cobrindo os dois formatos de config, a precedência
  entre eles, o clamp de valores inválidos e o parsing/plano do digest.

### Alterado

- O arquivo antigo `~/.config/tabelaradar/config` continua sendo lido quando
  não existe `config.toml`, com um aviso apontando pro caminho novo — nenhuma
  instalação existente quebra. Criado o `config.toml`, ele vence sozinho.
