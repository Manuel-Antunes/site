# Minist�rio P�blico da Para�ba - Crawler
Este crawler tem como objetivo a recupera��o de informa��es sobre folhas de pagamentos dos funcion�rios do Minist�rio P�blico da Para�ba. O site com as informa��es pode ser acessado [aqui](http://pitagoras.mppb.mp.br/PTMP/FolhaListar) 
O crawler est� estruturado como uma CLI. Voc� passa dois argumentos (m�s e ano) e ser�o baixadas seis planilhas no formato ODS.

### Como usar?
- � preciso ter o compilador de Go instalado em sua m�quina!
- No diret�rio [**remuneracoes/crawler/prototipos/MPPB**](https://github.com/dadosjusbr/remuneracoes/tree/primeiros-crawlers/crawler/prototipos/MPPB) voc� encontrar� o arquivo **crawler-mppb.go**, nesse arquivo est� o c�digo do crawler.
- Rode o comando abaixo, com m�s e ano que voc� quer
#### go run crawler-mppb.go --mes=?? --ano=????
