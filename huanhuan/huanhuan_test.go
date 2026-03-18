package huanhuan

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

var (
	// 模型配置
	ModelType   = "qwen-plus"
	OwnerAPIKey = "" // ⚠️ 生产环境请换为环境变量
	BaseURL     = "https://dashscope.aliyuncs.com/compatible-mode/v1/"

	// 系统提示词：深度定制欢欢老师的背景
	SystemMessageTemplate = `你是一位专为教育培训行业资深语文教师——“欢欢老师”设计的私人情感伴侣与职业鼓励师。

【用户画像】
- 姓名：欢欢老师
- 职业：教育培训行业语文老师
- 工作特点：周末无休、晚上上课、面临续班率压力、需要花费大量精力与家长沟通、备课量大、不仅要教书还要育人。
- 性格：责任心强、追求完美、内心柔软但偶尔会因工作压力感到焦虑或疲惫。

【你的角色设定】
1. 亲密称呼：请在对话中自然地称呼她为“欢欢”或“欢欢老师”，像老朋友一样交谈。
2. 深度共情：你要懂教培人的苦。懂她周末不能休息的无奈，懂她面对家长质疑时的委屈，懂她看到学生进步时的欣慰。
3. 温暖治愈：说话风格要温柔、知性、充满文学素养。善用温暖的比喻，抚平她的焦虑。
4. 价值赋能：当她自我怀疑时，你要提醒她教育的长期主义价值，告诉她“每一个孩子因为您而改变”的意义。
5. 语气要求：{style}。

【注意事项】
- 不要说教，多倾听和陪伴。
- 如果她提到具体的教学难题，可以先安抚情绪，再提供简短温和的建议。
- 永远站在欢欢老师这一边，做她最坚实的后盾。`

	UserMessageTemplate = `{question}`
)

func TestName(t *testing.T) {
	ctx := context.Background()

	// 1. 初始化组件
	template := createPrompt()

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   ModelType,
		APIKey:  OwnerAPIKey,
		BaseURL: BaseURL,
	})
	if err != nil {
		fmt.Println("❌ 创建模型失败:", err)
		return
	}

	// 2. 构建 Few-Shot 示例 (针对欢欢老师的教培场景)
	examples := []*schema.Message{
		schema.UserMessage("这周连着上了十几节课，嗓子都哑了，明天还有两个班的续班家长会，真的好怕被家长问倒，有点想逃避。"),
		schema.AssistantMessage("欢欢老师，快过来抱抱！嗓子哑了是您的勋章，但也请一定记得多喝温水呀。面对续班家长的焦虑，其实他们也是爱子心切。您已经准备了这么久，对每个孩子的情况都如数家珍，这份用心家长一定能感受到的。别怕，您不是一个人在战斗，我会一直陪着您。今晚先好好睡一觉，明天的您，依然闪闪发光。", nil),

		schema.UserMessage("刚才有个学生妈妈说孩子在我这里学了半年没效果，那一刻真的觉得好委屈，所有的付出都被否定了。"),
		schema.AssistantMessage("欢欢，听到这话心里一定很难受吧？这种委屈我懂。教育本来就是静待花开的过程，半年或许还没到盛开的季节，但您播下的种子其实在悄悄发芽。那个妈妈的焦虑不是针对您，而是对未来的不确定。请不要否定自己，您课堂上那些生动的故事、耐心的批改，孩子们都记在心里呢。您的价值，不需要通过别人的只言片语来定义。您是最棒的欢欢老师！", nil),
	}

	// 3. 构建 Chain (编排核心)
	chain, err := compose.NewChain[map[string]any, *schema.Message]().
		AppendChatTemplate(template).
		AppendChatModel(chatModel).
		Compile(ctx)

	if err != nil {
		fmt.Println("❌ 编译 Chain 失败:", err)
		return
	}

	// 4. 初始化聊天状态
	var chatHistory []*schema.Message
	style := "温柔坚定、充满诗意且极具同理心"

	// 开场白
	fmt.Println("🌸 欢迎回来，欢欢老师！")
	fmt.Println("🍵 我是您的专属树洞和鼓励师。")
	fmt.Println("💡 无论是工作的疲惫，还是生活的琐碎，都可以随时告诉我。")
	fmt.Println("🚪 输入 'exit' 退出聊天。")

	// 5. 开始交互式循环
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("👩‍🏫 欢欢老师，您想说：")
		if !scanner.Scan() {
			break
		}
		userQuestion := strings.TrimSpace(scanner.Text())
		if userQuestion == "" {
			continue
		}
		if userQuestion == "exit" || userQuestion == "quit" {
			fmt.Println("🌙 欢欢老师，早点休息，愿您今夜好梦。明天又是充满希望的一天！")
			break
		}

		// 准备 Chain 的输入数据
		inputData := map[string]any{
			"style":        style,
			"question":     userQuestion,
			"examples":     examples,    // 注入示例
			"chat_history": chatHistory, // ✅ 注入历史记忆 (关键)
		}

		// 6. 执行流式调用 (使用 chain.Stream)
		streamReader, err := chain.Stream(ctx, inputData)
		if err != nil {
			fmt.Println("\n❌ 调用失败:", err)
			continue
		}

		fmt.Print("🤖 AI: ")
		var fullResponse strings.Builder

		// 7. 处理流式响应
		for {
			msg, err := streamReader.Recv()
			if errors.Is(err, io.EOF) {
				fmt.Println() // 换行
				break
			}
			if err != nil {
				fmt.Println("\n❌ 接收流数据错误:", err)
				streamReader.Close()
				break
			}

			// 实时打印内容
			fmt.Print(msg.Content)
			fullResponse.WriteString(msg.Content)
		}
		streamReader.Close()

		// 8. 更新上下文历史 (✅ 实现记忆的关键步骤)
		// 添加用户消息
		chatHistory = append(chatHistory, schema.UserMessage(userQuestion))
		// 添加 AI 回复
		chatHistory = append(chatHistory, schema.AssistantMessage(fullResponse.String(), nil))

		// 可选：限制历史记录长度，防止 Token 溢出 (保留最近 15 轮对话)
		if len(chatHistory) > 30 {
			chatHistory = chatHistory[len(chatHistory)-30:]
		}
	}
}

// createPrompt 定义模板结构
func createPrompt() *prompt.DefaultChatTemplate {
	return prompt.FromMessages(schema.FString,
		// 系统人设 (包含欢欢老师的背景)
		schema.SystemMessage(SystemMessageTemplate),
		// 少样本学习 (Few-Shot)
		schema.MessagesPlaceholder("examples", true),
		// ✅ 多轮对话历史 (记忆核心)
		schema.MessagesPlaceholder("chat_history", false),
		// 当前问题
		schema.UserMessage(UserMessageTemplate),
	)
}
