# peer-discovery Specification

## Purpose
Permite que instâncias do OmniDesk se descubram dinamicamente na rede local (LAN) e estabeleçam relações de confiança mútua através de um pareamento seguro com PIN de 6 dígitos.

## Requirements

### Requirement: Local Peer Announcement and Discovery
The system SHALL / O nó do OmniDesk DEVE anunciar sua presença na rede local via mDNS e escutar por outros nós ativos na mesma rede sem requerer configuração manual de endereços IP.

#### Scenario: Anúncio de novo nó na rede
- **WHEN** o serviço OmniDesk é iniciado em uma máquina
- **THEN** ele publica um registro mDNS com nome do dispositivo, identificador único e porta de comunicação local

#### Scenario: Descoberta de nós existentes
- **WHEN** outros nós do OmniDesk estão ativos na mesma sub-rede
- **THEN** o nó local identifica automaticamente seus identificadores, nomes e endereços de rede dentro de 5 segundos

### Requirement: PIN-Based Mutual Device Pairing
The system SHALL / O sistema DEVE exigir autorização explícita através de um PIN numérico de 6 dígitos para autenticar a conexão entre dois dispositivos desconhecidos pela primeira vez.

#### Scenario: Solicitação e validação de pareamento bem-sucedida
- **WHEN** o usuário inicia um pareamento com um dispositivo descoberto
- **THEN** um código numérico de 6 dígitos idêntico é apresentado em ambos os dispositivos e, após confirmação explícita em ambos, o dispositivo é adicionado à lista de confiança mútua

#### Scenario: Rejeição de pareamento
- **WHEN** uma solicitação de pareamento não for confirmada ou for explicitamente rejeitada no dispositivo de destino
- **THEN** a sessão de pareamento é abortada imediatamente e nenhum acesso a clipboard ou arquivos é concedido

### Requirement: Trusted Devices Persistence
The system SHALL / O sistema DEVE persistir a lista de dispositivos pareados com segurança, permitindo conexões automáticas subsequentes sem novo pareamento.

#### Scenario: Reconexão de dispositivo confiável
- **WHEN** um dispositivo previamente pareado reaparece na rede local
- **THEN** a conexão mútua é restabelecida automaticamente utilizando os tokens criptográficos já persistidos
