import http from "k6/http";
import { check, sleep } from "k6";
 
export let options = {
 stages: [
    { duration: "30s", target: 20 },
    { duration: "1m30s", target: 10 },
    { duration: "20s", target: 5 },
    { duration: "1m", target: 50 },
    { duration: "30s", target: 10 },
    { duration: "30s", target: 100 },
  ]
};
 
export default function() {
  var url = getUrl()
 
  var params = {headers: { "Content-Type": "application/json" }}
  let res = http.get(url, params);
  check(res, {
    "status was 200": (r) => r.status == 200,
  });
  sleep(1);
}

function getUrl(){
  return "http://192.168.56.1:8080/quote"
}