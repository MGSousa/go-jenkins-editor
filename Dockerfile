FROM alpine

ARG JK_URL
ARG JK_PORT
ARG JK_USER
ARG JK_PASS
ARG JK_JOBSP

ENV APP_ENV="production"
ENV PORT=$JK_PORT
ENV USR=$JK_USER
ENV PWRD=$JK_PASS
ENV URL=$JK_URL
ENV PREFIX=$JK_JOBSP

WORKDIR /app

COPY ./dist/go-jenkins-editor_linux_amd64/go-jenkins-editor .

CMD ["sh", "-c", "./go-jenkins-editor -port=$PORT -username=$USR -password=$PWRD -jenkinsUrl=$URL -jobsPrefix=$PREFIX"]

EXPOSE $PORT