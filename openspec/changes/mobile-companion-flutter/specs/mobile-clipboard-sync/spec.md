# mobile-clipboard-sync Specification

## Purpose
Sincroniza texto e URLs da área de transferência de forma bidirecional entre o aplicativo mobile (Flutter) e os computadores desktop pareados na rede local, respeitando as políticas de privacidade do iOS e Android.

## Requirements

### Requirement: Mobile to Desktop Clipboard Transmission
The system SHALL / O aplicativo mobile DEVE permitir a transmissão intencional do conteúdo da área de transferência local para os computadores pareados e online.

#### Scenario: Envio manual de 1-toque
- **WHEN** o usuário toca no botão "Enviar Clipboard" no card de um computador pareado
- **THEN** o texto atual da área de transferência do smartphone é lido e transmitido via HTTP `POST /api/v1/clipboard` para o computador alvo
- **AND** o computador destino atualiza sua área de transferência do sistema operacional imediatamente

#### Scenario: Sincronização automática no retorno ao app
- **WHEN** o usuário abre ou retorna ao aplicativo OmniDesk no celular e a opção de auto-sync estiver habilitada
- **THEN** o aplicativo detecta o novo texto na área de transferência e transmite para os computadores pareados online

### Requirement: Desktop to Mobile Clipboard Reception
The system SHALL / O aplicativo mobile DEVE receber textos transmitidos pelos computadores pareados quando conectado na rede local e atualizar o clipboard do sistema operacional.

#### Scenario: Recepção de texto remoto no celular
- **WHEN** o computador pareado copia um texto e dispara o broadcast para o IP do smartphone
- **THEN** o aplicativo mobile recebe a carga HTTP autenticada
- **AND** injeta o texto na área de transferência do celular (se em primeiro plano) ou armazena no histórico com notificação local

### Requirement: Mobile Anti-Echo Loop Prevention
The system SHALL / O aplicativo mobile DEVE manter controle de deduplicação por hash SHA-256 para evitar loops de retransmissão de texto recebido.

#### Scenario: Injeção de texto recebido sem eco
- **WHEN** o aplicativo mobile escreve no clipboard um texto vindo de um computador
- **THEN** o hash desse texto é registrado em cache temporário
- **AND** nenhum evento subsequente de envio retransmite esse mesmo texto de volta para a rede
