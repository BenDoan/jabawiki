FROM debian:latest

RUN apt-get update
RUN apt-get install -q -y python3 python3-pip vim

RUN pip3 install gunicorn

ADD jabawiki /jabawiki
WORKDIR /jabawiki

VOLUME /tmp/jabawiki

EXPOSE 8080
#CMD ["python3", "jabawiki.py"]
CMD ["/usr/local/bin/gunicorn", "-b", "0.0.0.0:8080", "--error-logfile", "-", "jabawiki:app"]
