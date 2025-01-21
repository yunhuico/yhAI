#!/bin/bash
way=$(cd `dirname $0` && pwd)
num=`eval cat ${way}/AllImages.json | jq ".images | length"`
echo "total ${num} images"

files=(/deployer/resources/config/docker-compose-user.yml /deployer/resources/config/docker-compose.yml /deployer/resources/marathon/marathon-lb.json /deployer/resources/marathon/marathon-linkercomponents.json /deployer/dcos_deploy.properties /docker/docker-compose.yaml)
file_num=${#files[@]}
repo=linkerrepository

echo "start to change image's tag in file"
echofile() {
    echo -en "\033[36m" ## blue
    echo $1
    echo -en "\033[0m" ## reset color
}

if [ $num -ne 0 ]
then
   for (( i = 0; i < ${file_num}; i++)); do
   file=${files[i]}
   suffix=`eval echo ${file} | cut -d '.' -f2`
   echofile "now start to change ${file} images tag"
      for (( j=0; j< $num; j++)); do
      image=`eval cat ${way}/AllImages.json | jq -r ".images[$j].name"`
      image_temp=`eval echo ${image} | cut -d ':' -f1`
      image_name=${image_temp}:
      result=`eval echo ${image} | grep "gpu"`
      if [ "$result" != "" ]
      then
         if grep -q ${repo}/${image_name} ${way}${file}
         then
            echo "start to change mesos gpu tag"
            sed -i 's#'${repo}'/'${image_name}'.*gpu#'${repo}'/'${image}'#' ${way}${file}
         fi
      elif [ "${image_name}" = "mesos-slave:" ]
      then
         if grep -q ${repo}/${image_name} ${way}${file}
         then
         echo "start to change mesos slave tag"
         sed -i '70,$s#'${repo}'/'${image_name}'.*#'${repo}'/'${image}'#g' ${way}${file}
         fi
      else
         if grep -q ${repo}/${image_name} ${way}${file}
         then
            echo "start to change ${image_temp} tag"
            if [ "${suffix}" == "json" ]
            then
            sed -i 's#'${repo}'/'${image_name}'.*#'${repo}'/'${image}'",#g' /${way}${file}
            else
            sed -i 's#'${repo}'/'${image_name}'.*#'${repo}'/'${image}'#g' ${way}${file}
            fi
         fi
      fi
      done
   done
else
echo "num is 0!"
fi




