# mobile-media-file-transfer Specification

## Purpose
Permite a transferência direta e de alta velocidade de fotos, vídeos e arquivos entre smartphones (iOS / Android) e computadores desktop na mesma rede local, sem limite de tamanho e sem passar por nuvem.

## Requirements

### Requirement: Photo and File Upload from Mobile to Desktop
The system SHALL / O aplicativo mobile DEVE permitir a seleção de fotos da galeria ou documentos do armazenamento local e transmiti-los em streaming para o computador pareado.

#### Scenario: Envio de foto da galeria para o computador
- **WHEN** o usuário seleciona uma ou mais fotos e escolhe o computador de destino no aplicativo mobile
- **THEN** o arquivo é enviado em streaming HTTP direto para o endpoint `/api/v1/files/upload?filename=...` do computador
- **AND** o computador salva o arquivo diretamente na pasta configurada (ex.: Downloads) e exibe notificação nativa com o nome do arquivo

#### Scenario: Envio de documento/arquivo genérico
- **WHEN** o usuário seleciona um arquivo (PDF, ZIP, DOCX) através do seletor de arquivos
- **THEN** o arquivo é enviado para o computador alvo com barra de progresso visual no aplicativo mobile

### Requirement: File Reception on Mobile from Desktop
The system SHALL / O aplicativo mobile DEVE receber arquivos enviados por computadores pareados e salvá-los nos locais apropriados do sistema operacional mobile.

#### Scenario: Recepção de imagem no celular
- **WHEN** um computador pareado envia uma imagem para o smartphone
- **THEN** o aplicativo mobile recebe o stream de dados e salva a imagem na Galeria de Fotos / Rolo da Câmera do dispositivo
- **AND** adiciona o item na Central de Transferências do app com miniatura e confirmação de sucesso

#### Scenario: Recepção de arquivo genérico no celular
- **WHEN** um computador pareado envia um documento não-imagem
- **THEN** o aplicativo mobile salva o arquivo no diretório de Documentos do aplicativo
- **AND** permite ao usuário abrir o documento diretamente ou compartilhá-lo via Share Sheet do sistema operacional
