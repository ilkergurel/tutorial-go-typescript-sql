// LEARN(ts.async.combinators): Promise.all keeps tuple types; allSettled results are a discriminated union.

interface User {
  id: number;
  name: string;
}
interface Order {
  id: number;
  total: number;
}

const getUser = async (id: number): Promise<User> => ({ id, name: "Ayşe" });
const getOrders = async (userId: number): Promise<Order[]> => [
  { id: 1, total: 120 },
  { id: 2, total: 80 },
];
const getAvatar = async (userId: number): Promise<string> => {
  throw new Error(`avatar service down for user ${userId}`);
};

// Independent calls in parallel; the tuple keeps User and Order[] apart.
const [user, orders] = await Promise.all([getUser(7), getOrders(7)]);
console.log(`${user.name}: ${orders.length} orders`);

// The page can live without an avatar: settle both, then narrow on status.
const [avatar, recent] = await Promise.allSettled([getAvatar(7), getOrders(7)]);
const avatarUrl = avatar.status === "fulfilled" ? avatar.value : "/default.png";
if (avatar.status === "rejected") {
  // reason is typed any: treat it like a catch variable and narrow it.
  const why = avatar.reason instanceof Error ? avatar.reason.message : "?";
  console.log(`avatar: ${avatarUrl} (${why})`);
}
console.log(`recent orders: ${recent.status}`);

// A deadline with race. The loser is not cancelled, so clear the timer.
function withTimeout<T>(work: Promise<T>, ms: number): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  const deadline = new Promise<never>((_, reject) => {
    timer = setTimeout(() => reject(new Error(`timed out after ${ms} ms`)), ms);
  });
  return Promise.race([work, deadline]).finally(() => clearTimeout(timer));
}

const slow = new Promise<string>((resolve) =>
  setTimeout(() => {
    console.log("slow call finished anyway; nobody is listening");
    resolve("report");
  }, 300),
);
try {
  console.log(await withTimeout(slow, 50));
} catch (e) {
  console.log(e instanceof Error ? e.message : String(e));
}

export {}; // module: top-level await allowed
