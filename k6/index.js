import http from "k6/http";
import { check, sleep } from "k6";
import { uuidv4 } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";

export const options = {
  scenarios: {
    contacts: {
      executor: "ramping-vus",
      startVUs: 1,
      stages: [
        { target:  10, duration:  "2s" },
        { target: 100, duration:  "0s" }, // instantly jump to 100 VUs
        { target: 100, duration: "28s" }, // continue with 100 VUs for 28 seconds
      ],
    },
  },
};

const userId = uuidv4();
const sessionId = uuidv4();

export default function () {
  const headers = { "Content-Type": "application/json" };

  const postResponse = http.post(
    `http://localhost:1323/message`,
    JSON.stringify({
      userId: userId,
      sessionId: sessionId,
      message: "Hello, World!",
    }),
    { headers: headers, tags: { name: "post_message" } },
  );
  check(postResponse, { "POST: /message: 200": (r) => r.status == 200 });

  const getResponse = http.get(
    `http://localhost:1323/message?userId=${userId}&sessionId=${sessionId}&timestamp=${postResponse.json().timestamp}`,
    { tags: { name: "get_message" } },
  );
  check(getResponse, { "GET:  /message: 200": (r) => r.status == 200 });

  sleep(0.1);
}
