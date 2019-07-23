#!/bin/bash
set -e

images=$(echo $IMAGES | tr " " "\n")

for image in $images
do
    pack build $image --buildpack heroku/go --publish

    # if $PUSH_IMAGE
    # then
    #     docker push $image
    # fi

done

