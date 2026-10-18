package framework.prototype

import samples.Kettle

class Early extends ModelSuite(Kettle.Model):
  object examples:
    test("early"):
      ()
  val touched = examples
