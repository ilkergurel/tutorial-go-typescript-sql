// LEARN(ts.objects-classes.spread): spread copies one level; a nested update copies every level it changes.

interface Address {
  city: string;
  district: string;
}
interface Customer {
  readonly id: string;
  name: string;
  address: Address;
  tags: string[];
}

const customer = (): Customer => ({
  id: "c1",
  name: "Ayşe",
  address: { city: "İstanbul", district: "Kadıköy" },
  tags: ["vip"],
});

// Naive: the copy is new, but its address is the caller's address object.
function moveNaive(c: Customer, city: string): Customer {
  const copy = { ...c };
  copy.address.city = city;
  return copy;
}
const a = customer();
const b = moveNaive(a, "İzmir");
console.log(`naive: ${a.address.city} / ${b.address.city}`);

// Functional: copy each object on the path to the change, share the rest.
function move(c: Customer, city: string): Customer {
  return { ...c, address: { ...c.address, city } };
}
const c = customer();
const d = move(c, "Ankara");
console.log(`functional: ${c.address.city} / ${d.address.city}`);
console.log(`shared tags: ${d.tags === c.tags}, shared address: ${d.address === c.address}`);

// Later properties win, so defaults go first.
const defaults = { pageSize: 20, locale: "tr-TR" };
const fromUrl = { pageSize: 50 };
console.log({ ...defaults, ...fromUrl }, { ...fromUrl, ...defaults });

// Arrays: build new arrays instead of push, splice and sort.
const tags = ["vip", "yeni", "bayi"];
console.log([...tags, "kurumsal"], tags.filter((t) => t !== "yeni"));
console.log(tags.with(1, "eski"), tags.toSorted(), tags);
