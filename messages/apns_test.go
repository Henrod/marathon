/*
 * Copyright (c) 2016 TFG Co <backend@tfgco.com>
 * Author: TFG Co <backend@tfgco.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
 * the Software, and to permit persons to whom the Software is furnished to do so,
 * subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
 * FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
 * COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
 * IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package messages_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/topfreegames/marathon/messages"
)

var _ = Describe("APNS Message", func() {
	Describe("Creating new message", func() {
		It("should return message", func() {
			aps := map[string]interface{}{"x": 1}
			m := map[string]interface{}{"y": 2}
			pushMetadata := map[string]interface{}{
				"a": "b",
			}
			msg := messages.NewAPNSMessage("deviceToken", 357, aps, m, pushMetadata)

			Expect(msg).NotTo(BeNil())
			Expect(msg.DeviceToken).To(Equal("deviceToken"))
			Expect(msg.PushExpiry).To(BeEquivalentTo(357))
			Expect(msg.Payload).NotTo(BeNil())
			Expect(msg.Payload.Aps).To(BeEquivalentTo(aps))
			Expect(msg.Payload.M).To(BeEquivalentTo(m))
			Expect(msg.Metadata).To(BeEquivalentTo(pushMetadata))
		})

		It("should return message with nil maps", func() {
			empty := map[string]interface{}{}
			msg := messages.NewAPNSMessage("deviceToken", 357, nil, nil, nil)

			Expect(msg).NotTo(BeNil())
			Expect(msg.DeviceToken).To(Equal("deviceToken"))
			Expect(msg.PushExpiry).To(BeEquivalentTo(357))
			Expect(msg.Payload).NotTo(BeNil())
			Expect(msg.Payload.Aps).To(BeEquivalentTo(empty))
			Expect(msg.Payload.M).To(BeEquivalentTo(empty))
			Expect(msg.Metadata).To(BeEquivalentTo(empty))
		})
	})
})
