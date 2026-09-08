## ADDED Requirements

### Requirement: Permissão de controle de input distinta do pareamento
O sistema SHALL tratar a permissão de controle de mouse/teclado entre dois dispositivos como um estado separado da lista de dispositivos pareados (`TrustedDevices`), não concedida automaticamente pelo pareamento.

#### Scenario: Pareamento não concede controle de input
- **WHEN** dois dispositivos concluem o pareamento (PIN de 6 dígitos) pela primeira vez
- **THEN** nenhum dos dois dispositivos SHALL ter permissão de controlar o input do outro até que a permissão de input seja concedida explicitamente

### Requirement: Permissão concedida por direção, independentemente
O sistema SHALL registrar a permissão de controle de input separadamente para cada direção entre um par de dispositivos: "dispositivo remoto pode me controlar" e "eu posso controlar o dispositivo remoto" são estados independentes.

#### Scenario: Autorizar apenas uma direção
- **WHEN** o dispositivo A autoriza o dispositivo B a controlá-lo, mas o dispositivo B não concede a mesma autorização de volta
- **THEN** o dispositivo B pode controlar o input do dispositivo A, mas o dispositivo A não pode controlar o input do dispositivo B

### Requirement: Concessão de permissão exige confirmação explícita nas duas pontas
O sistema SHALL exigir uma ação explícita de aprovação no dispositivo que concede a permissão (o dispositivo que será controlado) antes de habilitar aquela direção de controle.

#### Scenario: Solicitação pendente até aprovação
- **WHEN** o dispositivo A solicita permissão para controlar o dispositivo B
- **THEN** a permissão permanece não concedida até que um usuário no dispositivo B aprove explicitamente a solicitação

#### Scenario: Solicitação recusada
- **WHEN** o dispositivo B recusa a solicitação de controle de input vinda do dispositivo A
- **THEN** a permissão não é concedida e o dispositivo A não pode abrir o canal de input com o dispositivo B

### Requirement: Revogação em cascata ao despareamento
O sistema SHALL remover automaticamente qualquer permissão de controle de input concedida em qualquer direção entre dois dispositivos quando o pareamento entre eles for removido.

#### Scenario: Remover dispositivo pareado remove também a permissão de input
- **WHEN** um usuário remove um dispositivo da lista de dispositivos confiáveis, e havia permissão de controle de input concedida com aquele dispositivo em qualquer direção
- **THEN** a permissão de controle de input associada a esse dispositivo é removida junto com o pareamento

### Requirement: Revogação manual independente do despareamento
O sistema SHALL permitir que um usuário revogue a permissão de controle de input em uma direção específica sem precisar remover o pareamento do dispositivo.

#### Scenario: Revogar controle sem desparear
- **WHEN** um usuário revoga a permissão de "dispositivo remoto pode me controlar" para um dispositivo específico pelo dashboard
- **THEN** o dispositivo remoto deixa de poder controlar o input local, mas o pareamento (clipboard/arquivo) continua funcionando normalmente
