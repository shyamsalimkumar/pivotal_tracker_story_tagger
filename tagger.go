package main
import (
    "os"
    "fmt"
    "net/http"
    "io/ioutil"
    "encoding/json"
    "sort"
    "time"
    "strconv"
    "io"
    "regexp"
)

type Label struct {
    Id int `json:"id"`
    ProjectId int `json:"project_id"`
    Kind string `json:"kind"`
    Name string `json:"name"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
    Id int `json:"id"`
    ProjectId int `json:"project_id"`
    Kind string `json:"kind"`
    Name string `json:"name"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    Estimate int `json:"estimate"`
    StoryType string `json:"story_type"`
    Description string `json:"description"`
    Url string `json:"url"`
    Labels []Label `json:"labels"`
}

type Messages []Message

func (slice Messages) Len() int {
    return len(slice)
}

func (slice Messages) Less(i, j int) bool {
    return slice[i].CreatedAt.Before(slice[j].CreatedAt);
}

func (slice Messages) Swap(i, j int) {
    slice[i], slice[j] = slice[j], slice[i]
}

type OutboundMessage struct {
    Name string `json:"name,omitempty"`
}

func getPivotalTrackerEnvVariables() (projectId, apiToken, storyPrefix string) {
    projectId = os.Getenv("PIVOTAL_TRACKER_PROJECT_ID")
    apiToken = os.Getenv("PIVOTAL_TRACKER_API_TOKEN")
    storyPrefix = os.Getenv("PIVOTAL_TRACKER_STORY_PREFIX")
    return
}

func makeRequest(client *http.Client, url string, method string, apiToken string, data io.Reader) (contents []byte,
    err error) {
    req, err := http.NewRequest(method, url, data)
    if err != nil {
        fmt.Println("[ERROR]:", err)
        os.Exit(1)
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-TrackerToken", apiToken)

    res, err := client.Do(req)
    if err != nil {
        fmt.Println("[ERROR]:", err)
        os.Exit(1)
    }
    defer res.Body.Close()

    contents, err = ioutil.ReadAll(res.Body)
    if err != nil {
        fmt.Println("[ERROR]:", err)
        os.Exit(1)
    }
    return
}

func getPrefix(prefixMap map[string]string, labels []Label) (string) {
    for _, label := range labels {
        fmt.Println("Checking", label, "in", prefixMap)
        if label, ok := prefixMap[label.Name]; ok {
            return label
        }
    }
    return ""
}

func main() {
    const (
        PIVOTAL_TRACKER_API = "https://www.pivotaltracker.com/services/v5/projects/"
        PAGES = 2
        RESULTS_PER_PAGE = 500
    )

    projectId, apiToken, storyPrefix := getPivotalTrackerEnvVariables()

    var stories Messages

    if projectId == "" || apiToken == "" || storyPrefix == "" {
        fmt.Println("[ERROR]: Please set PIVOTAL_TRACKER_PROJECT_ID, PIVOTAL_TRACKER_API_TOKEN and " +
            "PIVOTAL_TRACKER_STORY_PREFIX")
        os.Exit(1)
    }

    client := &http.Client{}
    baseUrl := PIVOTAL_TRACKER_API + projectId + "/"

    for i := 0; i < PAGES; i++ {
        listUrl := baseUrl + "stories?limit=" + strconv.Itoa(RESULTS_PER_PAGE) + "&offset=" + strconv.Itoa(RESULTS_PER_PAGE * i)

        contents, err := makeRequest(client, listUrl, "GET", apiToken, nil)
        if err != nil {
            fmt.Println("[ERROR]:", err)
            os.Exit(1)
        }

        var storiesFromJson Messages
        err = json.Unmarshal(contents, &storiesFromJson)
        if err != nil {
            fmt.Println("[ERROR]:", err)
            os.Exit(1)
        }
        stories = append(stories, storiesFromJson...)
    }

    sort.Sort(stories)

    keyMap := make(map[string]string)
    keyMap["CMN"] = "commons"
    keyMap["SEN"] = "sensor endpoints"
    keyMap["API"] = "api"
    keyMap["SPK"] = "spark"
    keyMap["WUI"] = "web ui"

    maxSeenStoryIdMap := make(map[string]int) // defaults are 0 so no need for initialization :D

    revKeyMap := make(map[string]string)
    for key, value := range keyMap {
        revKeyMap[value] = key
    }

//    Building regex
    regex := "("
    for key, _ := range keyMap {
        regex += key + "|"
    }
    regex = regex[:len(regex) - 1] + ")-(\\d+):"

    fmt.Println("Regex:", regex)

    re, err := regexp.Compile(regex)
    if err != nil {
        fmt.Println("[ERROR]:", err)
        os.Exit(1)
    }

    var newlyTaggedStoryCount int

    for _, story := range stories {
        pivotalId := story.Id
        storyName := story.Name

        matches := re.FindStringSubmatch(storyName)
        if len(matches) > 0 {
            matchedTag := matches[1]
            stringyStoryId := matches[2]
            storyId, err := strconv.Atoi(stringyStoryId)
            if err != nil {
                fmt.Println("[ERROR]:", err)
                os.Exit(1)
            }
            if maxSeenStoryIdMap[matchedTag] < storyId {
                maxSeenStoryIdMap[matchedTag] = storyId
            }
            fmt.Println("Skipping story with existing ID:", storyName)
            continue
        }

        label := getPrefix(revKeyMap, story.Labels)
        if label == "" {
            fmt.Println("[ERROR]:", "Could not match label to known keys")
            continue
        }
        newlyTaggedStoryCount += 1
        maxSeenStoryIdMap[label] += 1

        newStoryName := label + "-" + strconv.Itoa(maxSeenStoryIdMap[label]) +": " + storyName
        fmt.Println("Adding new ID to story:", newStoryName)

        updateUrl := baseUrl + "stories/" + strconv.Itoa(pivotalId)
        putData := &OutboundMessage{Name: newStoryName}
        b, err := json.Marshal(putData)
        if err != nil {
            fmt.Println("[ERROR]:", err)
            os.Exit(1)
        }
        fmt.Println(updateUrl)
        fmt.Println(string(b))

//        contents, err := makeRequest(client, updateUrl, "PUT", apiToken, bytes.NewBuffer(b))
//        if err != nil {
//            fmt.Println("[ERROR]:", err)
//            os.Exit(1)
//        }
//        fmt.Println(string(contents))
    }

    fmt.Printf("Tagged %d new stories out of %d total", newlyTaggedStoryCount, len(stories))
}
