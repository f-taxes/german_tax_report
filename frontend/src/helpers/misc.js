export const shorten = str => {
  if (str.length > 8) {
    return str.slice(0, 4) + '...' + str.slice(str.length - 4);
  } else if (str.length > 5) {
    return str.slice(0, 5) + '...';
  } else {
    return str;
  }
}