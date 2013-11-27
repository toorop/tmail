# Format du mail

###Objet javascript jasonifié stocke sous :
/toorop@tmail.io.inbox/msgID

###Les pièces jointes sont stockées sous :
/toorop@tmail.io.attachments/msgId.contentId

###Les sources brutes
/toorop@tmail.io.raw/msgId

###Les labels
/toorop@tmail.io.data/

*   /label_labelName
    + {msg:120, unseen:12}

*   /mbsize
    + {size:1221222333}

## Les champs Json
### mail.json
*   labels
    + inbox (dans la boite de reception principale)
    + important
    + todo
    + trash
    + sent
    + replied
    + forwarded
    + spam
    + ...

*   seen : bool

*   threadId

*   prev

*   next

*   headers :  contient tous les headers

*   text : le message au format text

*   html : le message au format html si présent

*   attachements
    + contentType": "application/pdf",
    + fileName": "Infra.pdf",
    + transferEncoding": "base64",
    + contentDisposition": "attachment",
    + generatedFileName": "Infra.pdf",
    + contentId": "6a338e40dd6f15b0f6ef793c5047ceeb@mailparser",
    + length": 20126,

