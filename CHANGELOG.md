# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Adicionado

- Config em TOML (`~/.config/tabelaradar/config.toml`), substituindo o formato
  de uma-linha-por-caminho. Além de `roots`/`exclude`, agora são configuráveis
  os arquivos de descrição, o dir de memória do Claude, as proporções de
  layout e o editor.
- Tecla `f5`: recarrega config.toml e keybindings sem reiniciar.
- Primeiros testes do repo, cobrindo os dois formatos de config, a precedência
  entre eles e o clamp de valores inválidos.

### Alterado

- O arquivo antigo `~/.config/tabelaradar/config` continua sendo lido quando
  não existe `config.toml`, com um aviso apontando pro caminho novo — nenhuma
  instalação existente quebra. Criado o `config.toml`, ele vence sozinho.
