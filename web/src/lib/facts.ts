/** Fact is one entry of a CountingFacts list: a label plus the whole
 *  number it counts up to, with an optional static suffix such as "+". */
export interface Fact {
  label: string
  target: number
  suffix?: string
}
