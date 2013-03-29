#!/usr/bin/env python

import email,json

mailFile = "/home/toorop/Projects/webmail.pro/dev/sample/mail_pdf.eml"

# On ouvre le mail et on le recupere comme objet
fp=open(mailFile,'r')
msg=email.message_from_file(fp)
fp.close()

# 
#i=0
#while msg.get_payload(i,True) :
#	print '\n\n %s' %msg.get_payload(i,True).as_string()
#	i=i+1


m=msg.get_payload(0,False)
#for m in t:
print m
#print 'toto %s' %m.get_content_type()

# Json
#jmsg=json.JSONEncoder().encode(msg)

#print jmsg

